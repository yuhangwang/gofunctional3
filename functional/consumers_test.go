// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package functional

import (
    "errors"
    "fmt"
    "testing"
)

var (
  consumerError = errors.New("consumer error.")
  evenFilterer = NewFilterer(func(ptr interface{}) error {
    p := ptr.(*int)
    if *p % 2 == 0 {
      return nil
    }
    return Skipped
  })
  squareFilterer = NewFilterer(func(ptr interface{}) error {
    p := ptr.(*int)
    *p = (*p) * (*p)
    return nil
  })

  plusOneFilterer = NewFilterer(func(ptr interface{}) error {
    p := ptr.(*int)
    *p = (*p) + 1
    return nil
  })
  
  oddFilterer = NewFilterer(func(ptr interface{}) error {
    p := ptr.(*int)
    if *p % 2 != 0 {
      return nil
    }
    return Skipped
  })

  squareFloat64IntMapper = NewMapper(func(src, dest interface{}) error {
    s := src.(*float64)
    d := dest.(*int)
    *d = int((*s) * (*s))
    return nil
  })

  float32Plus1Float64Mapper = NewMapper(func(src, dest interface{}) error {
    s := src.(*float32)
    d := dest.(*float64)
    *d = float64(*s + 1.0)
    return nil
  })
)

func TestMapConsumer(t *testing.T) {
  consumer := &intConsumer{}
  mconsumer := MapConsumer(
      consumer,
      squareFloat64IntMapper,
      new(float64))
  stream := NewStreamFromValues([]float64{1.733, 2.237, 3.163}, nil)
  doConsume(
      t,
      mconsumer,
      stream,
      nil)
  if output := fmt.Sprintf("%v", consumer.results); output != "[3 5 10]" {
    t.Errorf("Expected [3 5 10] got %v", output)
  }
}

func TestNestedMapConsumer(t *testing.T) {
  consumer := &intConsumer{}
  mconsumer := MapConsumer(
      MapConsumer(
          consumer,
          squareFloat64IntMapper,
          new(float64)),
      float32Plus1Float64Mapper,
      new(float32))
  stream := NewStreamFromValues([]float32{0.733, 1.237, 2.163}, nil)
  doConsume(
      t,
      mconsumer,
      stream,
      nil)
  if output := fmt.Sprintf("%v", consumer.results); output != "[3 5 10]" {
    t.Errorf("Expected [3 5 10] got %v", output)
  }
}

func TestNestedFilterConsumer(t *testing.T) {
  consumer := &intConsumer{}
  fconsumer := FilterConsumer(
      FilterConsumer(
          FilterConsumer(
              consumer,
              evenFilterer,
          ),
          plusOneFilterer,
      ),
      squareFilterer,
  )
  stream := NewStreamFromValues([]int{2, 3, 5, 7}, nil)
  doConsume(
      t,
      fconsumer,
      stream,
      nil)
  if output := fmt.Sprintf("%v", consumer.results); output != "[10 26 50]" {
    t.Errorf("Expected [10 26 50] got %v", output)
  }
}

func TestCompositeConsumer(t *testing.T) {
  ec := &intConsumer{}
  oc := &intConsumer{}
  consumer := CompositeConsumer(
      new(int),
      nil,
      FilterConsumer(ec, evenFilterer),
      FilterConsumer(oc, oddFilterer))
  doConsume(
      t,
      ModifyConsumer(consumer, func(s Stream) Stream { return Slice(s, 0, 5)}),
      Count(),
       nil)
  if output := fmt.Sprintf("%v", ec.results); output != "[0 2 4]" {
    t.Errorf("Expected [0 2 4] got %v", output)
  }
  if output := fmt.Sprintf("%v", oc.results); output != "[1 3]" {
    t.Errorf("Expected [1 3] got %v", output)
  }
}

func TestCompositeConsumerError(t *testing.T) {
  ec := &intConsumer{}
  oc := ConsumerFunc(func(s Stream) error { return consumerError })
  consumer := CompositeConsumer(
      new(int),
      nil,
      ec,
      oc)
  doConsume(t, consumer, Slice(Count(), 0, 5), consumerError)
}

func TestModifyConsumerStreamError(t *testing.T) {
  s := &streamCloseChecker{Count(), &simpleCloseChecker{}}
  var slice *streamCloseChecker
  f := func(s Stream) Stream {
    slice = &streamCloseChecker{Slice(s, 0, 5), &simpleCloseChecker{}}
    return slice
  }
  mc := ModifyConsumer(
      ConsumerFunc(func(s Stream) error { return consumerError}), f)
  doConsume(t, mc, s, consumerError)
  verifyCloseCalled(t, slice, true)
  verifyCloseCalled(t, s, false)
}

func TestModifyConsumerStreamAutoClose(t *testing.T) {
  s := &streamCloseChecker{Count(), &simpleCloseChecker{}}
  var slice *streamCloseChecker
  f := func(s Stream) Stream {
    slice = &streamCloseChecker{Slice(s, 0, 5), &simpleCloseChecker{}}
    return slice
  }
  mc := ModifyConsumer(NilConsumer(), f)
  doConsume(t, mc, s, nil)
  verifyCloseCalled(t, slice, true)
  verifyCloseCalled(t, s, false)
}

func TestModifyConsumerStreamAutoCloseError(t *testing.T) {
  s := &streamCloseChecker{Count(), &simpleCloseChecker{}}
  var slice *streamCloseChecker
  f := func(s Stream) Stream {
    slice = &streamCloseChecker{Slice(s, 0, 5), &simpleCloseChecker{closeError: closeError}}
    return slice
  }
  mc := ModifyConsumer(NilConsumer(), f)
  doConsume(t, mc, s, closeError)
  verifyCloseCalled(t, slice, true)
  verifyCloseCalled(t, s, false)
}

func TestConsumersNormal(t *testing.T) {
  s := Slice(Count(), 0, 5)
  ec := &intConsumer{}
  oc := &intConsumer{}
  errors := MultiConsume(
      s,
      new(int),
      nil,
      FilterConsumer(ec, evenFilterer),
      FilterConsumer(oc, oddFilterer))
  if len(errors) != 2 || errors[0] != nil || errors[1] != nil {
    t.Error("Expected no errors.")
  }
  if output := fmt.Sprintf("%v", ec.results); output != "[0 2 4]" {
    t.Errorf("Expected [0 2 4] got %v", output)
  }
  if output := fmt.Sprintf("%v", oc.results); output != "[1 3]" {
    t.Errorf("Expected [1 3] got %v", output)
  }
}

func TestConsumersEndEarly(t *testing.T) {
  s := Slice(Count(), 0, 5)
  ec := &intConsumer{}
  oc := &intConsumer{}
  nc := &noNextConsumer{}
  errors := MultiConsume(
      s,
      new(int),
      nil,
      nc,
      FilterConsumer(ec, evenFilterer),
      FilterConsumer(oc, oddFilterer))

  if len(errors) != 3 || errors[0] != nil || errors[1] != nil || errors[2] != nil {
    t.Errorf("Expected no errors, got %v %v %v", errors[0], errors[1], errors[2])
  }
  if output := fmt.Sprintf("%v", ec.results); output != "[0 2 4]" {
    t.Errorf("Expected [0 2 4] got %v", output)
  }
  if output := fmt.Sprintf("%v", oc.results); output != "[1 3]" {
    t.Errorf("Expected [1 3] got %v", output)
  }
  if !nc.completed {
    t.Error("MultiConsume returned before child consumers completed.")
  }
}

func TestMultiConsumeZeroOrOne(t *testing.T) {
  errors := MultiConsume(Count(), new(int), nil)
  if len(errors) != 0 {
    t.Errorf("Expected MultiConsume to return empty slice")
  }
  consumer := &intConsumer{}
  errors = MultiConsume(Slice(Count(), 0, 3), new(int), nil, consumer)
  if len(errors) != 1 || errors[0] != nil {
    t.Error("Expected nil error.")
  }
  if output := fmt.Sprintf("%v", consumer.results); output != "[0 1 2]" {
    t.Errorf("Expected [0 1 2] got %v", output)
  }
  ec := ConsumerFunc(func(s Stream) error { return consumerError })
  errors = MultiConsume(Count(), new(int), nil, ec)
  if len(errors) != 1 || errors[0] != consumerError {
    t.Error("Expected consumerError error.")
  }
}

func TestComositeConsumerZeroOrOne(t *testing.T) {
  if CompositeConsumer(
      new(int),
      nil) != NilConsumer() {
    t.Error("Expected composing zero consumers to be the Nil consumer.")
  }
  consumer := &intConsumer{}
  if CompositeConsumer(
      new(int),
      nil,
      consumer) != consumer {
    t.Errorf("Expected composing one consumer to be that consumer.")
  }
}

func TestReadPastEndConsumer(t *testing.T) {
  s := Slice(Count(), 0, 5)
  rc1 := &readPastEndConsumer{}
  rc2 := &readPastEndConsumer{}
  MultiConsume(s, new(int), nil, rc1, rc2)
  if !rc1.completed || !rc2.completed {
    t.Error("MultiConsume returned before child consumers completed.")
  }
}

type intConsumer struct {
  results []int
}

func (ic *intConsumer) Consume(s Stream) (err error) {
  ic.results, err = toIntArray(s)
  if err == Done {
    err = nil
  }
  return
}

type readPastEndConsumer struct {
  completed bool
}

func (c *readPastEndConsumer) Consume(s Stream) (err error) {
  toIntArray(s)
  var x int
  for i := 0; i < 10; i++ {
    s.Next(&x)
  }
  c.completed = true
  return
}

type noNextConsumer struct {
  completed bool
}

func (nc *noNextConsumer) Consume(s Stream) (err error) {
  nc.completed = true
  return
}

func doConsume(
    t *testing.T,
    c Consumer,
    s Stream,
    expectedError error) {
  if err := c.Consume(s); err != expectedError {
    t.Errorf("Expected %v, got %v", expectedError, err)
  }
}

