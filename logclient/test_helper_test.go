package logclient

// Allow logClient consumer to be modified, but only in tests (since the name of this file ends in "...test.go").
func (lc *logClient) SetConsumer(consumer Consumer) {
	lc.consumer = consumer
}

type ConsumerSetter interface {
	SetConsumer(consumer Consumer)
}
