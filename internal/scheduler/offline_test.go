package scheduler

import (
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestRequestNeverSent(t *testing.T) {
	dns := fmt.Errorf("instagram request failed: %w", &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host", IsNotFound: true}})
	timeout := fmt.Errorf("publish: %w", &net.OpError{Op: "read", Err: errors.New("i/o timeout")})
	if !requestNeverSent(dns) || requestNeverSent(timeout) || requestNeverSent(errors.New("status 400")) {
		t.Fatal("only dial/DNS failures may be retried without risking a double post")
	}
	chrome := errors.New("headless post failed (load x.com): page load error net::ERR_INTERNET_DISCONNECTED (GraphQL attempt before: lookup x.com: no such host)")
	if !requestNeverSent(chrome) {
		t.Fatal("headless Chrome offline error must be retried")
	}
}
