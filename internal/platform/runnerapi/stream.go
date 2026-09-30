package runnerapi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
)

// Stream caps (t3 §5.3, L14): enforced by DecodeJob as bytes arrive, so an oversize frame is
// refused from its length prefix, before its payload is read.
const (
	MaxJobJSONBytes   = 1 << 20  // job.json ≤ 1 MiB
	MaxCaseInputBytes = 8 << 20  // each case ≤ 8 MiB
	MaxInputsBytes    = 16 << 20 // all inputs ≤ 16 MiB
	MaxCases          = 512      // ≤ 512 cases
)

// frameHeaderBytes is the u32 big-endian length prefix.
const frameHeaderBytes = 4

// MaxBodyBytes bounds a whole POST /v1/jobs body: every frame at its cap plus the prefixes.
const MaxBodyBytes = frameHeaderBytes + MaxJobJSONBytes + MaxInputsBytes + frameHeaderBytes*MaxCases

// Caps are DecodeJob's limits. Tests may lower them; the runner uses DefaultCaps.
type Caps struct {
	MaxJobJSON   int64
	MaxCaseInput int64
	MaxInputs    int64
	MaxCases     int
}

// DefaultCaps are the contract caps.
func DefaultCaps() Caps {
	return Caps{MaxJobJSON: MaxJobJSONBytes, MaxCaseInput: MaxCaseInputBytes, MaxInputs: MaxInputsBytes, MaxCases: MaxCases}
}

// StreamError is a contract error in the job stream (an HTTP 400).
type StreamError struct {
	Reason string
}

func (e *StreamError) Error() string { return "runner job stream: " + e.Reason }

func streamErr(format string, a ...any) error {
	return &StreamError{Reason: fmt.Sprintf(format, a...)}
}

// IsStreamError reports whether err is a StreamError.
func IsStreamError(err error) bool {
	var se *StreamError
	return errors.As(err, &se)
}

// CheckContentType accepts exactly MediaTypeJob with v=1.
func CheckContentType(ct string) error {
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return streamErr("content type: %v", err)
	}
	if mt != MediaTypeJob {
		return streamErr("content type %q, want %s", mt, MediaTypeJob)
	}
	if v := params["v"]; v != StreamVersion {
		return streamErr("stream version %q, want %s", v, StreamVersion)
	}
	if len(params) != 1 {
		return streamErr("unexpected content-type parameters")
	}
	return nil
}

// EncodeJob writes the POST /v1/jobs body: job.json, then one frame per case input in Cases
// order. inputs[i] must be exactly job.Cases[i].Size bytes. It enforces the default caps, so
// judge can't build a body the runner refuses for size.
func EncodeJob(w io.Writer, job *Job, inputs [][]byte) error {
	if job == nil {
		return streamErr("nil job")
	}
	if len(inputs) != len(job.Cases) {
		return streamErr("%d inputs for %d cases", len(inputs), len(job.Cases))
	}
	if len(job.Cases) > MaxCases {
		return streamErr("%d cases, cap %d", len(job.Cases), MaxCases)
	}
	hdr, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	if len(hdr) > MaxJobJSONBytes {
		return streamErr("job.json is %d bytes, cap %d", len(hdr), MaxJobJSONBytes)
	}
	var total int64
	for i, in := range inputs {
		if int64(len(in)) != job.Cases[i].Size {
			return streamErr("case %d: input is %d bytes, Size says %d", i, len(in), job.Cases[i].Size)
		}
		if len(in) > MaxCaseInputBytes {
			return streamErr("case %d: input is %d bytes, cap %d", i, len(in), MaxCaseInputBytes)
		}
		total += int64(len(in))
		if total > MaxInputsBytes {
			return streamErr("inputs exceed %d bytes", MaxInputsBytes)
		}
	}
	bw := bufio.NewWriter(w)
	if err := writeFrame(bw, hdr); err != nil {
		return err
	}
	for _, in := range inputs {
		if err := writeFrame(bw, in); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func writeFrame(w io.Writer, b []byte) error {
	var h [frameHeaderBytes]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(b)))
	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

// readFrameLen reads one length prefix. A clean EOF before any byte returns io.EOF.
func readFrameLen(r io.Reader) (int64, error) {
	var h [frameHeaderBytes]byte
	n, err := io.ReadFull(r, h[:])
	if err != nil {
		if n == 0 && errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		return 0, streamErr("truncated frame header")
	}
	return int64(binary.BigEndian.Uint32(h[:])), nil
}

// DecodeJob reads a POST /v1/jobs body. It enforces caps as bytes arrive — each length prefix
// is checked before its payload is read — decodes job.json strictly (unknown fields are an
// error), and requires exactly one frame per case whose length equals the case's Size, then
// EOF. It does not run ValidateJob.
func DecodeJob(r io.Reader, caps Caps) (*Job, [][]byte, error) {
	n, err := readFrameLen(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, streamErr("empty body")
		}
		return nil, nil, err
	}
	if n > caps.MaxJobJSON {
		return nil, nil, streamErr("job.json is %d bytes, cap %d", n, caps.MaxJobJSON)
	}
	hdr := make([]byte, n)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, nil, streamErr("truncated job.json")
	}
	job, err := decodeJobJSON(hdr)
	if err != nil {
		return nil, nil, err
	}
	if len(job.Cases) > caps.MaxCases {
		return nil, nil, streamErr("%d cases, cap %d", len(job.Cases), caps.MaxCases)
	}
	inputs := make([][]byte, len(job.Cases))
	var total int64
	for i := range job.Cases {
		n, err := readFrameLen(r)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, nil, streamErr("%d case frames, want %d", i, len(job.Cases))
			}
			return nil, nil, err
		}
		if n > caps.MaxCaseInput {
			return nil, nil, streamErr("case %d: frame is %d bytes, cap %d", i, n, caps.MaxCaseInput)
		}
		if total+n > caps.MaxInputs {
			return nil, nil, streamErr("inputs exceed %d bytes", caps.MaxInputs)
		}
		if n != job.Cases[i].Size {
			return nil, nil, streamErr("case %d: frame is %d bytes, Size says %d", i, n, job.Cases[i].Size)
		}
		total += n
		in := make([]byte, n)
		if _, err := io.ReadFull(r, in); err != nil {
			return nil, nil, streamErr("case %d: truncated frame", i)
		}
		inputs[i] = in
	}
	// Exact frame count: the body must end here.
	var one [1]byte
	if k, err := io.ReadFull(r, one[:]); k != 0 || !errors.Is(err, io.EOF) {
		return nil, nil, streamErr("trailing bytes after the last case frame")
	}
	return job, inputs, nil
}

// decodeJobJSON is strict: unknown fields and trailing data are errors.
func decodeJobJSON(b []byte) (*Job, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var job Job
	if err := dec.Decode(&job); err != nil {
		return nil, streamErr("job.json: %v", err)
	}
	if dec.More() {
		return nil, streamErr("job.json: trailing data")
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, streamErr("job.json: trailing data")
	}
	return &job, nil
}
