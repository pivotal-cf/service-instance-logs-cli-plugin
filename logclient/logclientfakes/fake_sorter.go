// This file was generated by counterfeiter
package logclientfakes

import (
	"sync"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient"
)

type FakeSorter struct {
	SortRecentStub        func(messages []*events.LogMessage) []*events.LogMessage
	sortRecentMutex       sync.RWMutex
	sortRecentArgsForCall []struct {
		messages []*events.LogMessage
	}
	sortRecentReturns struct {
		result1 []*events.LogMessage
	}
	sortRecentReturnsOnCall map[int]struct {
		result1 []*events.LogMessage
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeSorter) SortRecent(messages []*events.LogMessage) []*events.LogMessage {
	var messagesCopy []*events.LogMessage
	if messages != nil {
		messagesCopy = make([]*events.LogMessage, len(messages))
		copy(messagesCopy, messages)
	}
	fake.sortRecentMutex.Lock()
	ret, specificReturn := fake.sortRecentReturnsOnCall[len(fake.sortRecentArgsForCall)]
	fake.sortRecentArgsForCall = append(fake.sortRecentArgsForCall, struct {
		messages []*events.LogMessage
	}{messagesCopy})
	fake.recordInvocation("SortRecent", []interface{}{messagesCopy})
	fake.sortRecentMutex.Unlock()
	if fake.SortRecentStub != nil {
		return fake.SortRecentStub(messages)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.sortRecentReturns.result1
}

func (fake *FakeSorter) SortRecentCallCount() int {
	fake.sortRecentMutex.RLock()
	defer fake.sortRecentMutex.RUnlock()
	return len(fake.sortRecentArgsForCall)
}

func (fake *FakeSorter) SortRecentArgsForCall(i int) []*events.LogMessage {
	fake.sortRecentMutex.RLock()
	defer fake.sortRecentMutex.RUnlock()
	return fake.sortRecentArgsForCall[i].messages
}

func (fake *FakeSorter) SortRecentReturns(result1 []*events.LogMessage) {
	fake.SortRecentStub = nil
	fake.sortRecentReturns = struct {
		result1 []*events.LogMessage
	}{result1}
}

func (fake *FakeSorter) SortRecentReturnsOnCall(i int, result1 []*events.LogMessage) {
	fake.SortRecentStub = nil
	if fake.sortRecentReturnsOnCall == nil {
		fake.sortRecentReturnsOnCall = make(map[int]struct {
			result1 []*events.LogMessage
		})
	}
	fake.sortRecentReturnsOnCall[i] = struct {
		result1 []*events.LogMessage
	}{result1}
}

func (fake *FakeSorter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.sortRecentMutex.RLock()
	defer fake.sortRecentMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeSorter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ logclient.Sorter = new(FakeSorter)