// Opt-in load benchmark.
//
// Spawns N WebSocket clients in parallel; each connects against an existing
// rekall-asr instance, replays a 60 s WAV in a loop, and records latency from
// the moment audio is sent to the moment a partial / final arrives. Reports
// P50/P95 at the end.
//
// Build:   cmake -DREKALL_ASR_BUILD_LOAD=ON
// Run:     ./asr_load --concurrency=8 --token=<jwt> --host=127.0.0.1 --port=8081
//
// The token must be minted by an external Go backend (or by hand using
// scripts/mint_dev_token.sh). The session it references must already be
// registered with the ASR service via gRPC.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <iostream>
#include <string>
#include <thread>
#include <vector>

#include <boost/asio.hpp>
#include <boost/beast.hpp>
#include <boost/beast/websocket.hpp>
#include <nlohmann/json.hpp>

namespace beast     = boost::beast;
namespace asio      = boost::asio;
namespace websocket = beast::websocket;
using tcp           = asio::ip::tcp;

namespace {

struct Args {
    int    concurrency = 4;
    std::string host = "127.0.0.1";
    std::string port = "8081";
    std::string token;
    std::string wav_path;
    int    duration_seconds = 30;
};

Args parse(int argc, char** argv) {
    Args a;
    for (int i = 1; i + 1 <= argc; ++i) {
        std::string s = argv[i];
        auto pull = [&](const char* k, std::string& dst) {
            if (s.rfind(k, 0) == 0) { dst = s.substr(std::strlen(k)); return true; }
            return false;
        };
        std::string v;
        if      (pull("--concurrency=", v))      a.concurrency      = std::stoi(v);
        else if (pull("--host=",        v))      a.host             = v;
        else if (pull("--port=",        v))      a.port             = v;
        else if (pull("--token=",       v))      a.token            = v;
        else if (pull("--wav=",         v))      a.wav_path         = v;
        else if (pull("--seconds=",     v))      a.duration_seconds = std::stoi(v);
    }
    return a;
}

std::vector<std::int16_t> load_pcm(const std::string& wav_path) {
    if (wav_path.empty()) {
        // Synthesise 1 s of a 1 kHz tone @ 16 kHz mono to keep the loop honest
        // when no WAV is provided.
        std::vector<std::int16_t> out(16000);
        for (std::size_t i = 0; i < out.size(); ++i) {
            double s = 8000.0 * std::sin(2.0 * 3.14159265358979 * 1000.0 * i / 16000.0);
            out[i] = static_cast<std::int16_t>(s);
        }
        return out;
    }
    std::ifstream f(wav_path, std::ios::binary);
    if (!f.good()) {
        std::fprintf(stderr, "cannot open %s\n", wav_path.c_str());
        std::exit(2);
    }
    f.seekg(44);  // skip canonical 44-byte WAV header
    std::vector<std::int16_t> out;
    std::int16_t s;
    while (f.read(reinterpret_cast<char*>(&s), sizeof(s))) out.push_back(s);
    return out;
}

}  // namespace

int main(int argc, char** argv) {
    Args args = parse(argc, argv);
    if (args.token.empty()) {
        std::fprintf(stderr, "missing --token=<jwt>\n");
        return 2;
    }

    auto pcm = load_pcm(args.wav_path);
    std::printf("loaded %zu samples (%.2f s) of audio\n",
                pcm.size(), static_cast<double>(pcm.size()) / 16000.0);

    std::vector<std::thread> clients;
    std::atomic<int> partials{0}, finals{0};
    std::vector<std::int64_t> first_partial_ms;
    first_partial_ms.resize(args.concurrency, -1);

    auto deadline = std::chrono::steady_clock::now() +
                    std::chrono::seconds(args.duration_seconds);

    for (int i = 0; i < args.concurrency; ++i) {
        clients.emplace_back([&, i]() {
            try {
                asio::io_context ioc;
                tcp::resolver resolver(ioc);
                auto endpoints = resolver.resolve(args.host, args.port);
                websocket::stream<tcp::socket> wsc(ioc);
                asio::connect(wsc.next_layer(), endpoints);
                wsc.set_option(websocket::stream_base::decorator(
                    [](websocket::request_type& req) {
                        req.set(beast::http::field::origin, "http://load-test");
                    }));
                wsc.handshake(args.host + ":" + args.port,
                              "/v1/asr/stream?token=" + args.token);

                auto start = std::chrono::steady_clock::now();
                std::thread reader([&]() {
                    beast::flat_buffer buf;
                    while (std::chrono::steady_clock::now() < deadline) {
                        beast::error_code ec;
                        wsc.read(buf, ec);
                        if (ec) break;
                        auto s = beast::buffers_to_string(buf.data());
                        buf.consume(buf.size());
                        try {
                            auto j = nlohmann::json::parse(s);
                            auto t = j.value("type", "");
                            if (t == "partial") {
                                if (first_partial_ms[i] < 0) {
                                    first_partial_ms[i] =
                                        std::chrono::duration_cast<std::chrono::milliseconds>(
                                            std::chrono::steady_clock::now() - start).count();
                                }
                                partials.fetch_add(1);
                            } else if (t == "final") {
                                finals.fetch_add(1);
                            }
                        } catch (...) {}
                    }
                });

                std::size_t cursor = 0;
                while (std::chrono::steady_clock::now() < deadline) {
                    constexpr std::size_t kFrame = 1600;  // 100 ms
                    if (cursor + kFrame > pcm.size()) cursor = 0;
                    wsc.binary(true);
                    wsc.write(asio::buffer(pcm.data() + cursor, kFrame * sizeof(std::int16_t)));
                    cursor += kFrame;
                    std::this_thread::sleep_for(std::chrono::milliseconds(100));
                }
                beast::error_code ec;
                wsc.close(websocket::close_code::normal, ec);
                reader.join();
            } catch (const std::exception& e) {
                std::fprintf(stderr, "client %d failed: %s\n", i, e.what());
            }
        });
    }
    for (auto& t : clients) t.join();

    auto valid = std::vector<std::int64_t>{};
    for (auto v : first_partial_ms) if (v >= 0) valid.push_back(v);
    std::sort(valid.begin(), valid.end());
    auto pct = [&](double p) -> std::int64_t {
        if (valid.empty()) return -1;
        auto i = static_cast<std::size_t>(p * (valid.size() - 1));
        return valid[i];
    };

    std::printf("\nresults (%d clients, %ds):\n", args.concurrency, args.duration_seconds);
    std::printf("  partials emitted: %d\n", partials.load());
    std::printf("  finals emitted:   %d\n", finals.load());
    std::printf("  first-partial latency: P50=%lldms P95=%lldms (n=%zu)\n",
                static_cast<long long>(pct(0.50)),
                static_cast<long long>(pct(0.95)),
                valid.size());
    return 0;
}
