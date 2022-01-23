package fsm_test

import (
	"errors"
	"testing"

	"github.com/quintans/fsm"
	"github.com/stretchr/testify/require"
)

func TestCompleteTripSuccess(t *testing.T) {
	r := require.New(t)
	payService := HealthyPayService{}

	trip := NewTrip()
	err := trip.Book("abc123")
	r.NoError(err)
	err = trip.Complete()
	r.NoError(err)
	err = trip.Pay(124, payService)
	r.NoError(err)
	r.Equal("abc123", trip.bookID)
	r.Equal(124, trip.fare)
	trip.state = "paid"
}

func TestCompleteTripFailure(t *testing.T) {
	r := require.New(t)
	payService := FaultyPayServiceImpl{}

	trip := NewTrip()
	err := trip.Book("abc123")
	r.NoError(err)
	err = trip.Complete()
	r.NoError(err)
	err = trip.Pay(123, payService)
	r.Error(err)
	r.Equal("abc123", trip.bookID)
	r.Equal(0, trip.fare)
	trip.state = "completed"
}

func TestCancelTripSuccess(t *testing.T) {
	r := require.New(t)
	payService := HealthyPayService{}

	trip := NewTrip()
	err := trip.Book("abc123")
	r.NoError(err)
	err = trip.Cancel(payService)
	r.NoError(err)
	r.Equal("abc123", trip.bookID)
	r.Equal(true, trip.cancelled)
	r.Equal(2, trip.fare)
	trip.state = "paid"
}

func TestCancelTripFail(t *testing.T) {
	r := require.New(t)
	payService := FaultyPayServiceImpl{}

	trip := NewTrip()
	err := trip.Book("abc123")
	r.NoError(err)
	err = trip.Cancel(payService)
	r.Error(err)
	r.Equal("abc123", trip.bookID)
	r.Equal(true, trip.cancelled)
	r.Equal(0, trip.fare)
	trip.state = "booked"
}

type PayService interface {
	Pay(amount int) error
}

type HealthyPayService struct{}

func (HealthyPayService) Pay(amount int) error {
	return nil
}

type FaultyPayServiceImpl struct{}

func (FaultyPayServiceImpl) Pay(amount int) error {
	return errors.New("invalid amount. it needs to be an even amount")
}

type Trip struct {
	sm        *fsm.StateMachine
	state     string
	cancelled bool
	bookID    string
	fare      int
}

func NewTrip() *Trip {
	t := &Trip{}
	sm := fsm.New()
	created := sm.AddState("created")
	booked := sm.AddState("booked", fsm.OnEnter(t.book))
	completed := sm.AddState("completed")
	cancelled := sm.AddState("cancelled", fsm.OnEvent(t.cancel))
	paid := sm.AddState("paid", fsm.OnEnter(t.paid))

	created.AddTransition("book", booked)
	booked.AddTransition("complete", completed)
	booked.AddTransition("cancel", cancelled)
	completed.AddTransition("pay", paid)
	cancelled.AddTransition("pay", paid)

	t.sm = sm
	t.state = created.Name()
	return t
}

type book struct {
	id string
}

func (book) Kind() interface{} {
	return "book"
}

type complete struct{}

func (complete) Kind() interface{} {
	return "complete"
}

type cancel struct {
	payService PayService
}

func (cancel) Kind() interface{} {
	return "cancel"
}

type pay struct {
	amount     int
	payService PayService
}

func (pay) Kind() interface{} {
	return "pay"
}

func (t *Trip) Book(bookID string) error {
	return t.fire(book{id: bookID})
}

func (t *Trip) book(c *fsm.Context) error {
	if t.bookID != "" {
		return nil
	}
	b := c.Data().(book)
	t.bookID = b.id
	return nil
}

func (t *Trip) Complete() error {
	return t.fire(complete{})
}

func (t *Trip) Cancel(payService PayService) error {
	return t.fire(cancel{
		payService: payService,
	})
}

func (t *Trip) cancel(c *fsm.Context) error {
	t.cancelled = true
	cncl := c.Data().(cancel)
	return c.Fire(pay{
		amount:     2,
		payService: cncl.payService,
	})
}

func (t *Trip) Pay(amount int, payService PayService) error {
	return t.fire(pay{
		amount:     amount,
		payService: payService,
	})
}

func (t *Trip) paid(c *fsm.Context) error {
	if t.fare != 0 {
		return nil
	}
	p := c.Data().(pay)
	err := p.payService.Pay(p.amount)
	if err != nil {
		return err
	}
	t.fare = p.amount
	return nil
}

func (t *Trip) fire(cmd fsm.Eventer) error {
	smi, err := t.sm.FromStateName(t.state)
	if err != nil {
		return err
	}
	err = smi.Fire(cmd)
	if err != nil {
		return err
	}
	t.state = smi.State().Name()
	return nil
}
