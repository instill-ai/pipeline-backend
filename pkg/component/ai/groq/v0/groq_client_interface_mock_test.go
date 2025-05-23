// Code generated by http://github.com/gojuno/minimock (v3.4.3). DO NOT EDIT.

package groq

//go:generate minimock -i github.com/instill-ai/pipeline-backend/pkg/component/ai/groq/v0.GroqClientInterface -o groq_client_interface_mock_test.go -n GroqClientInterfaceMock -p groq

import (
	_ "embed"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// GroqClientInterfaceMock implements GroqClientInterface
type GroqClientInterfaceMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcChat          func(c1 ChatRequest) (c2 ChatResponse, err error)
	funcChatOrigin    string
	inspectFuncChat   func(c1 ChatRequest)
	afterChatCounter  uint64
	beforeChatCounter uint64
	ChatMock          mGroqClientInterfaceMockChat
}

// NewGroqClientInterfaceMock returns a mock for GroqClientInterface
func NewGroqClientInterfaceMock(t minimock.Tester) *GroqClientInterfaceMock {
	m := &GroqClientInterfaceMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.ChatMock = mGroqClientInterfaceMockChat{mock: m}
	m.ChatMock.callArgs = []*GroqClientInterfaceMockChatParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mGroqClientInterfaceMockChat struct {
	optional           bool
	mock               *GroqClientInterfaceMock
	defaultExpectation *GroqClientInterfaceMockChatExpectation
	expectations       []*GroqClientInterfaceMockChatExpectation

	callArgs []*GroqClientInterfaceMockChatParams
	mutex    sync.RWMutex

	expectedInvocations       uint64
	expectedInvocationsOrigin string
}

// GroqClientInterfaceMockChatExpectation specifies expectation struct of the GroqClientInterface.Chat
type GroqClientInterfaceMockChatExpectation struct {
	mock               *GroqClientInterfaceMock
	params             *GroqClientInterfaceMockChatParams
	paramPtrs          *GroqClientInterfaceMockChatParamPtrs
	expectationOrigins GroqClientInterfaceMockChatExpectationOrigins
	results            *GroqClientInterfaceMockChatResults
	returnOrigin       string
	Counter            uint64
}

// GroqClientInterfaceMockChatParams contains parameters of the GroqClientInterface.Chat
type GroqClientInterfaceMockChatParams struct {
	c1 ChatRequest
}

// GroqClientInterfaceMockChatParamPtrs contains pointers to parameters of the GroqClientInterface.Chat
type GroqClientInterfaceMockChatParamPtrs struct {
	c1 *ChatRequest
}

// GroqClientInterfaceMockChatResults contains results of the GroqClientInterface.Chat
type GroqClientInterfaceMockChatResults struct {
	c2  ChatResponse
	err error
}

// GroqClientInterfaceMockChatOrigins contains origins of expectations of the GroqClientInterface.Chat
type GroqClientInterfaceMockChatExpectationOrigins struct {
	origin   string
	originC1 string
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmChat *mGroqClientInterfaceMockChat) Optional() *mGroqClientInterfaceMockChat {
	mmChat.optional = true
	return mmChat
}

// Expect sets up expected params for GroqClientInterface.Chat
func (mmChat *mGroqClientInterfaceMockChat) Expect(c1 ChatRequest) *mGroqClientInterfaceMockChat {
	if mmChat.mock.funcChat != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by Set")
	}

	if mmChat.defaultExpectation == nil {
		mmChat.defaultExpectation = &GroqClientInterfaceMockChatExpectation{}
	}

	if mmChat.defaultExpectation.paramPtrs != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by ExpectParams functions")
	}

	mmChat.defaultExpectation.params = &GroqClientInterfaceMockChatParams{c1}
	mmChat.defaultExpectation.expectationOrigins.origin = minimock.CallerInfo(1)
	for _, e := range mmChat.expectations {
		if minimock.Equal(e.params, mmChat.defaultExpectation.params) {
			mmChat.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmChat.defaultExpectation.params)
		}
	}

	return mmChat
}

// ExpectC1Param1 sets up expected param c1 for GroqClientInterface.Chat
func (mmChat *mGroqClientInterfaceMockChat) ExpectC1Param1(c1 ChatRequest) *mGroqClientInterfaceMockChat {
	if mmChat.mock.funcChat != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by Set")
	}

	if mmChat.defaultExpectation == nil {
		mmChat.defaultExpectation = &GroqClientInterfaceMockChatExpectation{}
	}

	if mmChat.defaultExpectation.params != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by Expect")
	}

	if mmChat.defaultExpectation.paramPtrs == nil {
		mmChat.defaultExpectation.paramPtrs = &GroqClientInterfaceMockChatParamPtrs{}
	}
	mmChat.defaultExpectation.paramPtrs.c1 = &c1
	mmChat.defaultExpectation.expectationOrigins.originC1 = minimock.CallerInfo(1)

	return mmChat
}

// Inspect accepts an inspector function that has same arguments as the GroqClientInterface.Chat
func (mmChat *mGroqClientInterfaceMockChat) Inspect(f func(c1 ChatRequest)) *mGroqClientInterfaceMockChat {
	if mmChat.mock.inspectFuncChat != nil {
		mmChat.mock.t.Fatalf("Inspect function is already set for GroqClientInterfaceMock.Chat")
	}

	mmChat.mock.inspectFuncChat = f

	return mmChat
}

// Return sets up results that will be returned by GroqClientInterface.Chat
func (mmChat *mGroqClientInterfaceMockChat) Return(c2 ChatResponse, err error) *GroqClientInterfaceMock {
	if mmChat.mock.funcChat != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by Set")
	}

	if mmChat.defaultExpectation == nil {
		mmChat.defaultExpectation = &GroqClientInterfaceMockChatExpectation{mock: mmChat.mock}
	}
	mmChat.defaultExpectation.results = &GroqClientInterfaceMockChatResults{c2, err}
	mmChat.defaultExpectation.returnOrigin = minimock.CallerInfo(1)
	return mmChat.mock
}

// Set uses given function f to mock the GroqClientInterface.Chat method
func (mmChat *mGroqClientInterfaceMockChat) Set(f func(c1 ChatRequest) (c2 ChatResponse, err error)) *GroqClientInterfaceMock {
	if mmChat.defaultExpectation != nil {
		mmChat.mock.t.Fatalf("Default expectation is already set for the GroqClientInterface.Chat method")
	}

	if len(mmChat.expectations) > 0 {
		mmChat.mock.t.Fatalf("Some expectations are already set for the GroqClientInterface.Chat method")
	}

	mmChat.mock.funcChat = f
	mmChat.mock.funcChatOrigin = minimock.CallerInfo(1)
	return mmChat.mock
}

// When sets expectation for the GroqClientInterface.Chat which will trigger the result defined by the following
// Then helper
func (mmChat *mGroqClientInterfaceMockChat) When(c1 ChatRequest) *GroqClientInterfaceMockChatExpectation {
	if mmChat.mock.funcChat != nil {
		mmChat.mock.t.Fatalf("GroqClientInterfaceMock.Chat mock is already set by Set")
	}

	expectation := &GroqClientInterfaceMockChatExpectation{
		mock:               mmChat.mock,
		params:             &GroqClientInterfaceMockChatParams{c1},
		expectationOrigins: GroqClientInterfaceMockChatExpectationOrigins{origin: minimock.CallerInfo(1)},
	}
	mmChat.expectations = append(mmChat.expectations, expectation)
	return expectation
}

// Then sets up GroqClientInterface.Chat return parameters for the expectation previously defined by the When method
func (e *GroqClientInterfaceMockChatExpectation) Then(c2 ChatResponse, err error) *GroqClientInterfaceMock {
	e.results = &GroqClientInterfaceMockChatResults{c2, err}
	return e.mock
}

// Times sets number of times GroqClientInterface.Chat should be invoked
func (mmChat *mGroqClientInterfaceMockChat) Times(n uint64) *mGroqClientInterfaceMockChat {
	if n == 0 {
		mmChat.mock.t.Fatalf("Times of GroqClientInterfaceMock.Chat mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmChat.expectedInvocations, n)
	mmChat.expectedInvocationsOrigin = minimock.CallerInfo(1)
	return mmChat
}

func (mmChat *mGroqClientInterfaceMockChat) invocationsDone() bool {
	if len(mmChat.expectations) == 0 && mmChat.defaultExpectation == nil && mmChat.mock.funcChat == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmChat.mock.afterChatCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmChat.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Chat implements GroqClientInterface
func (mmChat *GroqClientInterfaceMock) Chat(c1 ChatRequest) (c2 ChatResponse, err error) {
	mm_atomic.AddUint64(&mmChat.beforeChatCounter, 1)
	defer mm_atomic.AddUint64(&mmChat.afterChatCounter, 1)

	mmChat.t.Helper()

	if mmChat.inspectFuncChat != nil {
		mmChat.inspectFuncChat(c1)
	}

	mm_params := GroqClientInterfaceMockChatParams{c1}

	// Record call args
	mmChat.ChatMock.mutex.Lock()
	mmChat.ChatMock.callArgs = append(mmChat.ChatMock.callArgs, &mm_params)
	mmChat.ChatMock.mutex.Unlock()

	for _, e := range mmChat.ChatMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.c2, e.results.err
		}
	}

	if mmChat.ChatMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmChat.ChatMock.defaultExpectation.Counter, 1)
		mm_want := mmChat.ChatMock.defaultExpectation.params
		mm_want_ptrs := mmChat.ChatMock.defaultExpectation.paramPtrs

		mm_got := GroqClientInterfaceMockChatParams{c1}

		if mm_want_ptrs != nil {

			if mm_want_ptrs.c1 != nil && !minimock.Equal(*mm_want_ptrs.c1, mm_got.c1) {
				mmChat.t.Errorf("GroqClientInterfaceMock.Chat got unexpected parameter c1, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
					mmChat.ChatMock.defaultExpectation.expectationOrigins.originC1, *mm_want_ptrs.c1, mm_got.c1, minimock.Diff(*mm_want_ptrs.c1, mm_got.c1))
			}

		} else if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmChat.t.Errorf("GroqClientInterfaceMock.Chat got unexpected parameters, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
				mmChat.ChatMock.defaultExpectation.expectationOrigins.origin, *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmChat.ChatMock.defaultExpectation.results
		if mm_results == nil {
			mmChat.t.Fatal("No results are set for the GroqClientInterfaceMock.Chat")
		}
		return (*mm_results).c2, (*mm_results).err
	}
	if mmChat.funcChat != nil {
		return mmChat.funcChat(c1)
	}
	mmChat.t.Fatalf("Unexpected call to GroqClientInterfaceMock.Chat. %v", c1)
	return
}

// ChatAfterCounter returns a count of finished GroqClientInterfaceMock.Chat invocations
func (mmChat *GroqClientInterfaceMock) ChatAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmChat.afterChatCounter)
}

// ChatBeforeCounter returns a count of GroqClientInterfaceMock.Chat invocations
func (mmChat *GroqClientInterfaceMock) ChatBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmChat.beforeChatCounter)
}

// Calls returns a list of arguments used in each call to GroqClientInterfaceMock.Chat.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmChat *mGroqClientInterfaceMockChat) Calls() []*GroqClientInterfaceMockChatParams {
	mmChat.mutex.RLock()

	argCopy := make([]*GroqClientInterfaceMockChatParams, len(mmChat.callArgs))
	copy(argCopy, mmChat.callArgs)

	mmChat.mutex.RUnlock()

	return argCopy
}

// MinimockChatDone returns true if the count of the Chat invocations corresponds
// the number of defined expectations
func (m *GroqClientInterfaceMock) MinimockChatDone() bool {
	if m.ChatMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.ChatMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.ChatMock.invocationsDone()
}

// MinimockChatInspect logs each unmet expectation
func (m *GroqClientInterfaceMock) MinimockChatInspect() {
	for _, e := range m.ChatMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to GroqClientInterfaceMock.Chat at\n%s with params: %#v", e.expectationOrigins.origin, *e.params)
		}
	}

	afterChatCounter := mm_atomic.LoadUint64(&m.afterChatCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.ChatMock.defaultExpectation != nil && afterChatCounter < 1 {
		if m.ChatMock.defaultExpectation.params == nil {
			m.t.Errorf("Expected call to GroqClientInterfaceMock.Chat at\n%s", m.ChatMock.defaultExpectation.returnOrigin)
		} else {
			m.t.Errorf("Expected call to GroqClientInterfaceMock.Chat at\n%s with params: %#v", m.ChatMock.defaultExpectation.expectationOrigins.origin, *m.ChatMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcChat != nil && afterChatCounter < 1 {
		m.t.Errorf("Expected call to GroqClientInterfaceMock.Chat at\n%s", m.funcChatOrigin)
	}

	if !m.ChatMock.invocationsDone() && afterChatCounter > 0 {
		m.t.Errorf("Expected %d calls to GroqClientInterfaceMock.Chat at\n%s but found %d calls",
			mm_atomic.LoadUint64(&m.ChatMock.expectedInvocations), m.ChatMock.expectedInvocationsOrigin, afterChatCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *GroqClientInterfaceMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockChatInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *GroqClientInterfaceMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *GroqClientInterfaceMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockChatDone()
}
