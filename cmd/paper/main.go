// Package main is the mock FIX counterparty.
package main

import (
	"os"

	"github.com/gbkr-com/exo/env"
	"github.com/quickfixgo/quickfix"
)

func main() {

	file, err := os.Open("settings.cfg")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	settings, err := quickfix.ParseSettings(file)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	store := quickfix.NewMemoryStoreFactory()
	log := quickfix.NewScreenLogFactory()

	app := NewApplication()

	acceptor, err := quickfix.NewAcceptor(app, store, settings, log)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	acceptor.Start()
	<-env.Signal()
	acceptor.Stop()

}
