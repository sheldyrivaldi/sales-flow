package hermes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// AgentTask is one fire-and-forget generation handed to the bridge. The bridge
// does the work in its own background and POSTs the result to CallbackURL, so
// the app never holds a long connection waiting for generation.
// AgentDocument adalah satu lampiran yang ikut dikirim bersama tugas.
type AgentDocument struct {
	Filename string
	Bytes    []byte
}

type AgentTask struct {
	Instruction    string
	JobID          string
	CallbackURL    string
	CallbackSecret string
	// Documents: BANYAK lampiran sekaligus. Bridge merentang tiap PDF menjadi
	// gambar per halaman lalu menggabungkan semuanya jadi satu pesan.
	Documents []AgentDocument
	// Filename/FileBytes: bentuk tunggal lama, masih dipakai jalur playbook.
	Filename  string
	FileBytes []byte
}

// AgentTaskRunner is an optional capability (like DocumentExtractor): fire a
// background agent task at the bridge and return as soon as it's accepted
// (HTTP 202). Kept OUT of the Client interface so the many minimal test stubs
// implementing Client aren't forced to add a method only this flow needs.
type AgentTaskRunner interface {
	RunAgentTask(ctx context.Context, task AgentTask) error
}

type wireAgentTaskReq struct {
	Instruction      string         `json:"instruction"`
	JobID            string         `json:"job_id"`
	CallbackURL      string         `json:"callback_url"`
	CallbackSecret   string         `json:"callback_secret,omitempty"`
	DocumentBase64   string         `json:"document_base64,omitempty"`
	DocumentFilename string         `json:"document_filename,omitempty"`
	Documents        []wireDocument `json:"documents,omitempty"`
}

// wireDocument adalah bentuk kawat satu lampiran pada payload agent-task.
type wireDocument struct {
	Base64   string `json:"base64"`
	Filename string `json:"filename"`
}

// RunAgentTask POSTs to /v1/agent-task. The bridge replies 202 immediately and
// runs the agent in its own background; this returns as soon as that ack
// arrives, so the caller never blocks on generation.
func (c *httpClient) RunAgentTask(ctx context.Context, task AgentTask) error {
	wireReq := wireAgentTaskReq{
		Instruction:    task.Instruction,
		JobID:          task.JobID,
		CallbackURL:    task.CallbackURL,
		CallbackSecret: task.CallbackSecret,
	}
	for _, d := range task.Documents {
		if len(d.Bytes) == 0 {
			continue
		}
		wireReq.Documents = append(wireReq.Documents, wireDocument{
			Base64:   base64.StdEncoding.EncodeToString(d.Bytes),
			Filename: d.Filename,
		})
	}
	if len(task.FileBytes) > 0 {
		wireReq.DocumentBase64 = base64.StdEncoding.EncodeToString(task.FileBytes)
		wireReq.DocumentFilename = task.Filename
	}
	payload, err := json.Marshal(wireReq)
	if err != nil {
		return fmt.Errorf("hermes agent-task: marshal: %w", err)
	}

	req, err := c.newReq(ctx, "POST", "/v1/agent-task", bytes.NewReader(payload), "", "")
	if err != nil {
		return fmt.Errorf("hermes agent-task: build request: %w", err)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("hermes agent-task: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readErr(resp)
	}
	return nil
}
