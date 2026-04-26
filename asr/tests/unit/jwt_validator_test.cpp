#include "rekall/asr/auth/jwt_validator.hpp"

#include <gtest/gtest.h>
#include <jwt-cpp/jwt.h>

#include <chrono>
#include <memory>
#include <variant>

using namespace rekall::asr::auth;
using namespace std::chrono_literals;

namespace {

constexpr const char* kSecret  = "0123456789abcdef0123456789abcdef0123456789ab";
constexpr const char* kAud     = "rekall-asr";
constexpr const char* kIss     = "rekall-backend";

std::string make_token(std::chrono::system_clock::time_point exp,
                       const std::string& sid  = "sid-123",
                       const std::string& jti  = "jti-1",
                       const std::string& aud  = kAud,
                       const std::string& iss  = kIss,
                       const std::string& sub  = "user-1",
                       std::chrono::system_clock::time_point nbf =
                           std::chrono::system_clock::now() - 1s,
                       const std::string& secret = kSecret) {
    return jwt::create()
        .set_issuer(iss)
        .set_audience(aud)
        .set_subject(sub)
        .set_id(jti)
        .set_payload_claim("sid", jwt::claim(sid))
        .set_payload_claim("cid", jwt::claim(std::string("call-1")))
        .set_payload_claim("model", jwt::claim(std::string("small.en")))
        .set_issued_at(std::chrono::system_clock::now())
        .set_not_before(nbf)
        .set_expires_at(exp)
        .sign(jwt::algorithm::hs256{secret});
}

JWTValidator make_validator() {
    auto cache = std::make_shared<JtiCache>(60s);
    return JWTValidator(kSecret, kAud, kIss, cache);
}

}  // namespace

TEST(JWTValidatorTest, AcceptsWellFormedToken) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() + 60s);
    auto r = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenClaims>(r));
    auto c = std::get<TokenClaims>(r);
    EXPECT_EQ(c.sid, "sid-123");
    EXPECT_EQ(c.cid, "call-1");
    EXPECT_EQ(c.sub, "user-1");
    EXPECT_EQ(c.jti, "jti-1");
}

TEST(JWTValidatorTest, RejectsBadSignature) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() + 60s,
                            "sid-1", "jti-bad-sig", kAud, kIss, "user-1",
                            std::chrono::system_clock::now() - 1s,
                            "another-secret-of-32-bytes-or-more!");
    auto r = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::InvalidSignature);
}

TEST(JWTValidatorTest, RejectsExpired) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() - 1s,
                            "sid-1", "jti-expired");
    auto r = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::Expired);
}

TEST(JWTValidatorTest, RejectsWrongAudience) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() + 60s,
                            "sid-1", "jti-aud", "another-aud");
    auto r = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::WrongAudience);
}

TEST(JWTValidatorTest, RejectsWrongIssuer) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() + 60s,
                            "sid-1", "jti-iss", kAud, "another-iss");
    auto r = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::WrongIssuer);
}

TEST(JWTValidatorTest, RejectsMalformed) {
    auto v = make_validator();
    auto r = v.validate("not.a.jwt");
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::Malformed);
}

TEST(JWTValidatorTest, RejectsReplay) {
    auto v = make_validator();
    auto token = make_token(std::chrono::system_clock::now() + 60s,
                            "sid-1", "jti-replay");
    auto r1 = v.validate(token);
    EXPECT_TRUE(std::holds_alternative<TokenClaims>(r1));
    auto r2 = v.validate(token);
    ASSERT_TRUE(std::holds_alternative<TokenError>(r2));
    EXPECT_EQ(std::get<TokenError>(r2), TokenError::JtiReplay);
}

TEST(JWTValidatorTest, EmptyTokenIsMalformed) {
    auto v = make_validator();
    auto r = v.validate("");
    ASSERT_TRUE(std::holds_alternative<TokenError>(r));
    EXPECT_EQ(std::get<TokenError>(r), TokenError::Malformed);
}

TEST(JWTValidatorTest, ReasonStringsAreStable) {
    EXPECT_EQ(reason(TokenError::InvalidSignature), "invalid_signature");
    EXPECT_EQ(reason(TokenError::Expired),          "expired");
    EXPECT_EQ(reason(TokenError::WrongAudience),    "wrong_audience");
    EXPECT_EQ(reason(TokenError::JtiReplay),        "jti_replay");
}
