#include "rekall/asr/auth/jwt_validator.hpp"

#include <jwt-cpp/jwt.h>

#include <chrono>
#include <stdexcept>
#include <string>
#include <variant>

namespace rekall::asr::auth {

std::string_view reason(TokenError e) noexcept {
    switch (e) {
        case TokenError::InvalidSignature: return "invalid_signature";
        case TokenError::Expired:          return "expired";
        case TokenError::NotYetValid:      return "not_yet_valid";
        case TokenError::WrongAudience:    return "wrong_audience";
        case TokenError::WrongIssuer:      return "wrong_issuer";
        case TokenError::UnknownSession:   return "unknown_session";
        case TokenError::JtiReplay:        return "jti_replay";
        case TokenError::Malformed:        return "malformed";
    }
    return "unknown";
}

JWTValidator::JWTValidator(std::string secret, std::string audience, std::string issuer,
                           std::shared_ptr<JtiCache> jti)
    : secret_(std::move(secret)),
      audience_(std::move(audience)),
      issuer_(std::move(issuer)),
      jti_(std::move(jti)) {}

std::variant<TokenClaims, TokenError> JWTValidator::validate(std::string_view token) const {
    if (token.empty()) return TokenError::Malformed;

    jwt::decoded_jwt<jwt::traits::kazuho_picojson> decoded =
        [&]() -> jwt::decoded_jwt<jwt::traits::kazuho_picojson> {
            try {
                return jwt::decode(std::string(token));
            } catch (const std::exception&) {
                throw TokenError::Malformed;
            }
        }();

    try {
        // Verify signature, alg, iss, aud, exp, nbf in one pass.
        auto verifier = jwt::verify()
                            .allow_algorithm(jwt::algorithm::hs256{secret_})
                            .with_issuer(issuer_)
                            .with_audience(audience_);
        verifier.verify(decoded);
    } catch (const jwt::error::token_verification_exception& e) {
        // Inspect what failed to surface a precise reason.
        const std::string what = e.what();
        if (what.find("signature") != std::string::npos)        return TokenError::InvalidSignature;
        if (what.find("expired") != std::string::npos ||
            what.find("exp") != std::string::npos)              return TokenError::Expired;
        if (what.find("nbf") != std::string::npos ||
            what.find("not yet") != std::string::npos)          return TokenError::NotYetValid;
        if (what.find("audience") != std::string::npos ||
            what.find("aud") != std::string::npos)              return TokenError::WrongAudience;
        if (what.find("issuer") != std::string::npos ||
            what.find("iss") != std::string::npos)              return TokenError::WrongIssuer;
        return TokenError::InvalidSignature;
    } catch (const std::exception&) {
        return TokenError::Malformed;
    }

    TokenClaims claims;
    try {
        claims.sub = decoded.get_subject();
        if (decoded.has_payload_claim("sid")) {
            claims.sid = decoded.get_payload_claim("sid").as_string();
        } else {
            return TokenError::Malformed;
        }
        if (decoded.has_payload_claim("cid")) {
            claims.cid = decoded.get_payload_claim("cid").as_string();
        }
        if (decoded.has_payload_claim("model")) {
            claims.model = decoded.get_payload_claim("model").as_string();
        }
        if (!decoded.has_payload_claim("jti")) return TokenError::Malformed;
        claims.jti = decoded.get_id();
        claims.exp = decoded.get_expires_at();
    } catch (const std::exception&) {
        return TokenError::Malformed;
    }

    // Single-use enforcement is the LAST step — only consume the jti once we
    // know the token is otherwise valid; that prevents an attacker poisoning
    // legitimate jti slots with garbage tokens.
    if (jti_ && !jti_->try_consume(claims.jti, claims.exp)) {
        return TokenError::JtiReplay;
    }

    return claims;
}

}  // namespace rekall::asr::auth
