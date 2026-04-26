package pb

// Custom gRPC codec for the rekall-asr service.
//
// The hand-maintained pb stubs in this package don't carry full
// protoreflect descriptors — `MessageInfo.Desc` is nil, which makes the
// default grpc-go proto codec panic inside protoimpl reflection (see
// `makeKnownFieldsFunc`). Building real descriptors without protoc is a
// pile of boilerplate, so we instead supply a codec that hand-encodes the
// specific request/response types this client actually exchanges with the
// ASR service, dispatched by Go type rather than by reflection.
//
// Coverage: StartSession, EndSession, GetSession, Health, ReloadModels,
// plus emptypb.Empty (used by Health). Streaming paths are not used by the
// backend; they panic with a clear error if invoked.

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protowire"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// CodecName is the gRPC codec identifier the backend dial forces. Anything
// other than "proto" so we don't collide with the registered default.
const CodecName = "rekall-asr-proto"

func init() {
	encoding.RegisterCodec(asrCodec{})
}

type asrCodec struct{}

// Codec returns the singleton custom gRPC codec used by the backend's ASR
// client. Pass to grpc.ForceCodec(...) when dialing.
func Codec() encoding.Codec { return asrCodec{} }

func (asrCodec) Name() string { return CodecName }

func (asrCodec) Marshal(v any) ([]byte, error) {
	switch m := v.(type) {
	case *StartSessionRequest:
		return marshalStartSessionRequest(m), nil
	case *EndSessionRequest:
		return marshalEndSessionRequest(m), nil
	case *GetSessionRequest:
		return marshalGetSessionRequest(m), nil
	case *emptypb.Empty:
		return []byte{}, nil
	case *ReloadModelsRequest:
		return marshalReloadModelsRequest(m), nil
	// Server-side responses (the backend doesn't run an ASR server, but we
	// keep these so a future in-process test harness wouldn't blow up).
	case *StartSessionResponse:
		return marshalStartSessionResponse(m), nil
	case *EndSessionResponse:
		return marshalEndSessionResponse(m), nil
	case *SessionInfo:
		return marshalSessionInfo(m), nil
	case *HealthResponse:
		return marshalHealthResponse(m), nil
	case *ReloadModelsResponse:
		return marshalReloadModelsResponse(m), nil
	default:
		return nil, fmt.Errorf("rekall-asr-proto: cannot marshal %T", v)
	}
}

func (asrCodec) Unmarshal(data []byte, v any) error {
	switch m := v.(type) {
	case *StartSessionResponse:
		return unmarshalStartSessionResponse(data, m)
	case *EndSessionResponse:
		return unmarshalEndSessionResponse(data, m)
	case *SessionInfo:
		return unmarshalSessionInfo(data, m)
	case *HealthResponse:
		return unmarshalHealthResponse(data, m)
	case *ReloadModelsResponse:
		return unmarshalReloadModelsResponse(data, m)
	case *emptypb.Empty:
		return nil
	// Client-side requests (server-side decode path; unused today).
	case *StartSessionRequest:
		return unmarshalStartSessionRequest(data, m)
	case *EndSessionRequest:
		return unmarshalEndSessionRequest(data, m)
	case *GetSessionRequest:
		return unmarshalGetSessionRequest(data, m)
	case *ReloadModelsRequest:
		return unmarshalReloadModelsRequest(data, m)
	default:
		return fmt.Errorf("rekall-asr-proto: cannot unmarshal into %T", v)
	}
}

// ─── Encoders ───────────────────────────────────────────────────────────────
//
// All encoders follow the proto3 rule: skip fields equal to the default value
// (empty string, zero number, nil message). Order doesn't matter on the wire,
// but we emit by ascending field number for determinism.

func marshalStartSessionRequest(m *StartSessionRequest) []byte {
	var b []byte
	if m.UserId != "" {
		b = appendString(b, 1, m.UserId)
	}
	if m.CallId != "" {
		b = appendString(b, 2, m.CallId)
	}
	if m.ModelId != "" {
		b = appendString(b, 3, m.ModelId)
	}
	if m.Language != "" {
		b = appendString(b, 4, m.Language)
	}
	if m.RequestedTokenTtlSeconds != 0 {
		b = appendVarint(b, 5, uint64(m.RequestedTokenTtlSeconds))
	}
	for k, v := range m.Metadata {
		// map<string,string> entries are length-delimited messages with
		// key=field 1 (string), value=field 2 (string).
		var entry []byte
		entry = appendString(entry, 1, k)
		entry = appendString(entry, 2, v)
		b = appendBytes(b, 6, entry)
	}
	return b
}

func marshalEndSessionRequest(m *EndSessionRequest) []byte {
	var b []byte
	if m.SessionId != "" {
		b = appendString(b, 1, m.SessionId)
	}
	return b
}

func marshalGetSessionRequest(m *GetSessionRequest) []byte {
	var b []byte
	if m.SessionId != "" {
		b = appendString(b, 1, m.SessionId)
	}
	return b
}

func marshalReloadModelsRequest(m *ReloadModelsRequest) []byte {
	var b []byte
	for _, e := range m.Entries {
		b = appendBytes(b, 1, marshalModelEntry(e))
	}
	return b
}

func marshalModelEntry(m *ModelEntry) []byte {
	var b []byte
	if m.Id != "" {
		b = appendString(b, 1, m.Id)
	}
	if m.Path != "" {
		b = appendString(b, 2, m.Path)
	}
	if m.Language != "" {
		b = appendString(b, 3, m.Language)
	}
	if m.NThreads != 0 {
		b = appendVarint(b, 4, uint64(m.NThreads))
	}
	if m.BeamSize != 0 {
		b = appendVarint(b, 5, uint64(m.BeamSize))
	}
	return b
}

func marshalStartSessionResponse(m *StartSessionResponse) []byte {
	var b []byte
	if m.SessionId != "" {
		b = appendString(b, 1, m.SessionId)
	}
	if m.ModelId != "" {
		b = appendString(b, 2, m.ModelId)
	}
	if m.SampleRate != 0 {
		b = appendVarint(b, 3, uint64(m.SampleRate))
	}
	if m.FrameFormat != "" {
		b = appendString(b, 4, m.FrameFormat)
	}
	if m.ExpiresAt != nil {
		b = appendBytes(b, 5, marshalTimestamp(m.ExpiresAt))
	}
	return b
}

func marshalEndSessionResponse(m *EndSessionResponse) []byte {
	var b []byte
	if m.FinalTranscript != "" {
		b = appendString(b, 1, m.FinalTranscript)
	}
	if m.FinalCount != 0 {
		b = appendVarint(b, 2, uint64(m.FinalCount))
	}
	return b
}

func marshalSessionInfo(m *SessionInfo) []byte {
	var b []byte
	if m.State != "" {
		b = appendString(b, 1, m.State)
	}
	if m.StartedAt != nil {
		b = appendBytes(b, 2, marshalTimestamp(m.StartedAt))
	}
	if m.LastActivityAt != nil {
		b = appendBytes(b, 3, marshalTimestamp(m.LastActivityAt))
	}
	if m.AudioSecondsProcessed != 0 {
		b = appendVarint(b, 4, uint64(m.AudioSecondsProcessed))
	}
	if m.PartialCount != 0 {
		b = appendVarint(b, 5, uint64(m.PartialCount))
	}
	if m.FinalCount != 0 {
		b = appendVarint(b, 6, uint64(m.FinalCount))
	}
	return b
}

func marshalHealthResponse(m *HealthResponse) []byte {
	var b []byte
	if m.Status != "" {
		b = appendString(b, 1, m.Status)
	}
	if m.Version != "" {
		b = appendString(b, 2, m.Version)
	}
	if m.UptimeSeconds != 0 {
		b = appendVarint(b, 3, m.UptimeSeconds)
	}
	for _, s := range m.LoadedModels {
		b = appendString(b, 4, s)
	}
	if m.ActiveSessions != 0 {
		b = appendVarint(b, 5, uint64(m.ActiveSessions))
	}
	if m.WorkerPoolSize != 0 {
		b = appendVarint(b, 6, uint64(m.WorkerPoolSize))
	}
	if m.WorkerPoolInUse != 0 {
		b = appendVarint(b, 7, uint64(m.WorkerPoolInUse))
	}
	return b
}

func marshalReloadModelsResponse(m *ReloadModelsResponse) []byte {
	var b []byte
	for _, s := range m.Loaded {
		b = appendString(b, 1, s)
	}
	for _, s := range m.Failed {
		b = appendString(b, 2, s)
	}
	return b
}

func marshalTimestamp(t *timestamppb.Timestamp) []byte {
	var b []byte
	if t.Seconds != 0 {
		b = appendVarint(b, 1, uint64(t.Seconds))
	}
	if t.Nanos != 0 {
		b = appendVarint(b, 2, uint64(int64(t.Nanos)))
	}
	return b
}

// ─── Decoders ───────────────────────────────────────────────────────────────

func unmarshalStartSessionResponse(data []byte, m *StartSessionResponse) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.SessionId = s
			data = data[n:]
		case 2:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.ModelId = s
			data = data[n:]
		case 3:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.SampleRate = int32(v)
			data = data[n:]
		case 4:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.FrameFormat = s
			data = data[n:]
		case 5:
			raw, n := consumeBytes(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			ts := &timestamppb.Timestamp{}
			if err := unmarshalTimestamp(raw, ts); err != nil {
				return err
			}
			m.ExpiresAt = ts
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalEndSessionResponse(data []byte, m *EndSessionResponse) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.FinalTranscript = s
			data = data[n:]
		case 2:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.FinalCount = uint32(v)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalSessionInfo(data []byte, m *SessionInfo) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.State = s
			data = data[n:]
		case 2:
			raw, n := consumeBytes(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			ts := &timestamppb.Timestamp{}
			if err := unmarshalTimestamp(raw, ts); err != nil {
				return err
			}
			m.StartedAt = ts
			data = data[n:]
		case 3:
			raw, n := consumeBytes(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			ts := &timestamppb.Timestamp{}
			if err := unmarshalTimestamp(raw, ts); err != nil {
				return err
			}
			m.LastActivityAt = ts
			data = data[n:]
		case 4:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.AudioSecondsProcessed = uint32(v)
			data = data[n:]
		case 5:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.PartialCount = uint32(v)
			data = data[n:]
		case 6:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.FinalCount = uint32(v)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalHealthResponse(data []byte, m *HealthResponse) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.Status = s
			data = data[n:]
		case 2:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.Version = s
			data = data[n:]
		case 3:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.UptimeSeconds = v
			data = data[n:]
		case 4:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.LoadedModels = append(m.LoadedModels, s)
			data = data[n:]
		case 5:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.ActiveSessions = uint32(v)
			data = data[n:]
		case 6:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.WorkerPoolSize = uint32(v)
			data = data[n:]
		case 7:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.WorkerPoolInUse = uint32(v)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalReloadModelsResponse(data []byte, m *ReloadModelsResponse) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.Loaded = append(m.Loaded, s)
			data = data[n:]
		case 2:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.Failed = append(m.Failed, s)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalTimestamp(data []byte, t *timestamppb.Timestamp) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			t.Seconds = int64(v)
			data = data[n:]
		case 2:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			t.Nanos = int32(v)
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

// Server-side decoders for completeness (no production caller hits them).

func unmarshalStartSessionRequest(data []byte, m *StartSessionRequest) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.UserId = s
			data = data[n:]
		case 2:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.CallId = s
			data = data[n:]
		case 3:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.ModelId = s
			data = data[n:]
		case 4:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.Language = s
			data = data[n:]
		case 5:
			v, n := consumeVarint(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.RequestedTokenTtlSeconds = uint32(v)
			data = data[n:]
		case 6:
			raw, n := consumeBytes(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			if m.Metadata == nil {
				m.Metadata = map[string]string{}
			}
			var k, val string
			for len(raw) > 0 {
				en, et, en2 := protowire.ConsumeTag(raw)
				if en2 < 0 {
					return protowire.ParseError(en2)
				}
				raw = raw[en2:]
				switch en {
				case 1:
					s, n := consumeString(raw, et)
					if n < 0 {
						return protowire.ParseError(n)
					}
					k = s
					raw = raw[n:]
				case 2:
					s, n := consumeString(raw, et)
					if n < 0 {
						return protowire.ParseError(n)
					}
					val = s
					raw = raw[n:]
				default:
					n := protowire.ConsumeFieldValue(en, et, raw)
					if n < 0 {
						return protowire.ParseError(n)
					}
					raw = raw[n:]
				}
			}
			m.Metadata[k] = val
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalEndSessionRequest(data []byte, m *EndSessionRequest) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.SessionId = s
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalGetSessionRequest(data []byte, m *GetSessionRequest) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		switch num {
		case 1:
			s, n := consumeString(data, typ)
			if n < 0 {
				return protowire.ParseError(n)
			}
			m.SessionId = s
			data = data[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalReloadModelsRequest(data []byte, m *ReloadModelsRequest) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}
		data = data[n:]
		if num != 1 {
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return protowire.ParseError(n)
			}
			data = data[n:]
			continue
		}
		// silent: ReloadModels is not exercised by the backend client path
		return errors.New("ReloadModels decode not implemented")
	}
	return nil
}

// ─── Wire helpers ───────────────────────────────────────────────────────────

func appendString(b []byte, num int, s string) []byte {
	b = protowire.AppendTag(b, protowire.Number(num), protowire.BytesType)
	b = protowire.AppendString(b, s)
	return b
}

func appendBytes(b []byte, num int, p []byte) []byte {
	b = protowire.AppendTag(b, protowire.Number(num), protowire.BytesType)
	b = protowire.AppendBytes(b, p)
	return b
}

func appendVarint(b []byte, num int, v uint64) []byte {
	b = protowire.AppendTag(b, protowire.Number(num), protowire.VarintType)
	b = protowire.AppendVarint(b, v)
	return b
}

func consumeString(data []byte, typ protowire.Type) (string, int) {
	if typ != protowire.BytesType {
		return "", -1
	}
	v, n := protowire.ConsumeString(data)
	return v, n
}

func consumeBytes(data []byte, typ protowire.Type) ([]byte, int) {
	if typ != protowire.BytesType {
		return nil, -1
	}
	v, n := protowire.ConsumeBytes(data)
	return v, n
}

func consumeVarint(data []byte, typ protowire.Type) (uint64, int) {
	if typ != protowire.VarintType {
		return 0, -1
	}
	v, n := protowire.ConsumeVarint(data)
	return v, n
}
