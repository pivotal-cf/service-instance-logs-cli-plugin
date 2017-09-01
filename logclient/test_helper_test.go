package logclient

// Allow logClient consumer to be modified, but only in tests (since the name of this file ends in "...test.go").
func (lc *logClient) SetConsumer(consumer Consumer) {
	lc.consumer = consumer
}

// Allow logClient sorter to be modified, but only in tests (since the name of this file ends in "...test.go").
func (lc *logClient) SetSorter(sorter Sorter) {
	lc.sorter = sorter
}

type FieldSetter interface {
	SetConsumer(consumer Consumer)
	SetSorter(sorter Sorter)
}

type BuildWithConsumer interface {
	BuildFromConsumer(cons Consumer) LogClient
}
