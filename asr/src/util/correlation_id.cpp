#include "rekall/asr/util/correlation_id.hpp"

#include <array>
#include <cctype>
#include <iomanip>
#include <random>
#include <sstream>
#include <string>
#include <string_view>

namespace rekall::asr::util {

namespace {

bool is_hex(char c) {
    return std::isxdigit(static_cast<unsigned char>(c)) != 0;
}

}  // namespace

bool looks_like_uuid_v4(std::string_view s) {
    if (s.size() != 36) return false;
    for (std::size_t i = 0; i < s.size(); ++i) {
        if (i == 8 || i == 13 || i == 18 || i == 23) {
            if (s[i] != '-') return false;
        } else if (!is_hex(s[i])) {
            return false;
        }
    }
    // Version nibble (char 14) must be '4'.
    if (s[14] != '4') return false;
    // Variant nibble (char 19) must be one of 8, 9, a, b.
    char v = static_cast<char>(std::tolower(static_cast<unsigned char>(s[19])));
    return v == '8' || v == '9' || v == 'a' || v == 'b';
}

std::string new_uuid_v4() {
    static thread_local std::mt19937_64 rng{std::random_device{}()};
    std::array<std::uint8_t, 16> b{};
    std::uint64_t hi = rng();
    std::uint64_t lo = rng();
    for (int i = 0; i < 8; ++i) b[i]     = static_cast<std::uint8_t>((hi >> (8 * i)) & 0xFF);
    for (int i = 0; i < 8; ++i) b[8 + i] = static_cast<std::uint8_t>((lo >> (8 * i)) & 0xFF);
    // Set version (4) and variant (10xx).
    b[6] = static_cast<std::uint8_t>((b[6] & 0x0F) | 0x40);
    b[8] = static_cast<std::uint8_t>((b[8] & 0x3F) | 0x80);

    std::ostringstream os;
    os << std::hex << std::setfill('0');
    for (std::size_t i = 0; i < b.size(); ++i) {
        if (i == 4 || i == 6 || i == 8 || i == 10) os << '-';
        os << std::setw(2) << static_cast<int>(b[i]);
    }
    return os.str();
}

std::string ensure(std::string_view incoming) {
    if (!incoming.empty()) return std::string(incoming);
    return new_uuid_v4();
}

}  // namespace rekall::asr::util
