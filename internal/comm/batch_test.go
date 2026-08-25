package comm

import (
	"errors"
	"testing"

	"pvcontrol/internal/inverter"
)

// failingTransport succeeds until FailAfter sends, then fails the next one.
type failingTransport struct {
	table      *inverter.Table
	sent       int
	FailAfter int
}

func (t *failingTransport) Open(invID string) (Handle, error) {
	return &failHandle{tr: t}, nil
}

type failHandle struct {
	tr *failingTransport
}

func (h *failHandle) ID() string { return "fail" }
func (h *failHandle) Write(payload []byte) error {
	if h.tr.sent >= h.tr.FailAfter {
		return errors.New("pvcontrol: simulated comm timeout")
	}
	return nil
}
func (h *failHandle) Read() ([]byte, error) {
	if h.tr.sent >= h.tr.FailAfter {
		return nil, errors.New("pvcontrol: simulated comm timeout")
	}
	h.tr.sent++
	ack := &inverter.Frame{Serial: "S1", Kind: inverter.FrameAck, Seq: 1, Payload: []byte("ok")}
	return EncodeFrame(ack), nil
}
func (h *failHandle) Close() error { return nil }

func enqueueBatch(t *testing.T, c *Client, invID, kind string, count int) {
	t.Helper()
	payloads := make([][]byte, count)
	for i := range payloads {
		payloads[i] = inverter.EncodePower(1000 + i)
	}
	if _, err := c.EnqueueBatch(invID, kind, payloads); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
}

// TestProcessBatchWritesFreshResults guards against the regression where the
// write-back used a stale prior-batch snapshot, regressing already-sent
// commands back to pending in the message table.
func TestProcessBatchWritesFreshResults(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	transport := &failingTransport{table: table, FailAfter: 1 << 30}
	c := NewClient(table, transport)

	enqueueBatch(t, c, "inv1", string(inverter.FrameLimit), 3)
	if _, err := c.ProcessPending("inv1"); err != nil {
		t.Fatalf("process pending: %v", err)
	}

	for seq := 1; seq <= 3; seq++ {
		res, err := table.MessageStatus("inv1", seq)
		if err != nil {
			t.Fatalf("status seq %d: %v", seq, err)
		}
		if res.Status != inverter.MessageSent {
			t.Fatalf("seq %d status = %s, want sent (fresh result must not be overwritten by stale snapshot)", seq, res.Status)
		}
	}
}

// TestProcessBatchPreservesBoundaryOnFailure verifies that messages which
// succeeded before a later failure (boundary commands) keep their sent state
// instead of being left pending.
func TestProcessBatchPreservesBoundaryOnFailure(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	// Succeed once, then fail: batch of 3 -> seq1 sent, seq2 failed, seq3 unprocessed.
	transport := &failingTransport{table: table, FailAfter: 1}
	c := NewClient(table, transport)

	enqueueBatch(t, c, "inv1", string(inverter.FrameLimit), 3)
	if _, err := c.ProcessPending("inv1"); err == nil {
		t.Fatal("process pending: want error from failed boundary message")
	}

	r1, _ := table.MessageStatus("inv1", 1)
	if r1.Status != inverter.MessageSent {
		t.Fatalf("boundary seq 1 status = %s, want sent (boundary command must be preserved)", r1.Status)
	}
	r2, _ := table.MessageStatus("inv1", 2)
	if r2.Status != inverter.MessageFailed {
		t.Fatalf("failed seq 2 status = %s, want failed", r2.Status)
	}
	r3, _ := table.MessageStatus("inv1", 3)
	if r3.Status != inverter.MessagePending {
		t.Fatalf("unprocessed seq 3 status = %s, want pending", r3.Status)
	}
}

// TestProcessBatchRetryRecover verifies that a successfully sent command does
// not regress to pending. This is the core of the reported issue: a sent
// command that was overwritten with a stale snapshot stays pending, so it is
// retried forever ("retry still fails"). With the fix, the sent result
// survives, PendingMessages no longer returns it, and there is nothing left
// to retry.
func TestProcessBatchRetryRecover(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	transport := &failingTransport{table: table, FailAfter: 1 << 30}
	c := NewClient(table, transport)

	enqueueBatch(t, c, "inv1", string(inverter.FrameLimit), 2)
	if _, err := c.ProcessPending("inv1"); err != nil {
		t.Fatalf("process pending: %v", err)
	}

	// Every command was sent; the table must reflect that so PendingMessages
	// drains the queue (nothing left to retry).
	for seq := 1; seq <= 2; seq++ {
		res, _ := table.MessageStatus("inv1", seq)
		if res.Status != inverter.MessageSent {
			t.Fatalf("seq %d status = %s, want sent (stale snapshot must not regress sent -> pending)", seq, res.Status)
		}
	}
	pending, err := c.table.PendingMessages("inv1")
	if err != nil {
		t.Fatalf("pending messages: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("retry still has %d pending messages (sent result did not stick, retry loop never ends)", len(pending))
	}
}

