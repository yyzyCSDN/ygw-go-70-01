package comm

func (c *Client) EnqueueBatch(invID string, kind string, payloads [][]byte) (int, error) {
	start := c.nextSeq()
	for index, payload := range payloads {
		if err := c.Enqueue(invID, start+index, kind, payload); err != nil {
			return index, err
		}
	}
	return len(payloads), nil
}

func (c *Client) ProcessPending(invID string) (int, error) {
	pending, err := c.table.PendingMessages(invID)
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	results, err := c.ProcessBatch(pending)
	return len(results), err
}
