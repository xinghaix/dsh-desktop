package main

import (
	"fmt"
	"os"
	"testing"
)

func TestAcceptDiscoverPrint(t *testing.T) {
	if os.Getenv("GO_ACCEPT_DISCOVER") != "1" {
		t.Skip("set GO_ACCEPT_DISCOVER=1")
	}
	rep := ProbeHostDiscovery()
	fmt.Println("===ACCEPT_DISCOVER_BEGIN===")
	fmt.Println(rep.Message)
	if rep.Hit != nil {
		fmt.Printf("HIT reason=%s path=%s\n", rep.Hit.Reason, rep.Hit.Path)
	} else {
		fmt.Println("HIT=(none)")
	}
	for _, c := range rep.Checked {
		fmt.Println("CHECKED", c)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		fmt.Println("BOOTSTRAP_ERR", err)
	} else {
		fmt.Println("BOOTSTRAP_CMD", cmd)
	}
	fmt.Println("===ACCEPT_DISCOVER_END===")
}
