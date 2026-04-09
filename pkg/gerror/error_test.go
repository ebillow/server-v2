package gerror

import (
	"errors"
	"fmt"
	"server/pkg/pb"
	"testing"
)

func CallFrom1() error {
	err := From(pb.ErrorCode_BadRequest, errors.New("id error"))
	return err
}

func CallFrom2() error {
	return CallFrom1()
}

func CallFrom3() error {
	return CallFrom2()
}

func TestFrom(t *testing.T) {
	err := CallFrom3()
	fmt.Printf("%+v\n", err)
}

func CallNew1() error {
	err := NewCode(pb.ErrorCode_BadRequest)
	return err
}

func CallNew() error {
	return CallNew1()
}

func TestNew(t *testing.T) {
	err := CallNew()
	fmt.Printf("%+v\n", err)
}

func TestNew2(t *testing.T) {
	err := New("cfg not found")
	fmt.Printf("%+v\n", err)
}

func TestWithStack(t *testing.T) {
	err := New("cfg not found")
	fmt.Printf("%+v\n", WithStack(err))
}

func TestWrap(t *testing.T) {
	err := CallNew1()
	fmt.Printf("%+v\n", Wrap(err, "call new error"))
}
