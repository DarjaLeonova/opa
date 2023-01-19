package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
	"log"
	"strconv"
	"time"
)

var ctx = context.Background()

//go:embed example.rego
var module string

var input = []map[string]interface{}{
	{
		"method": "GET",
		"path":   []string{"users", "bobby78"},
		"customer": map[string]interface{}{
			"login":    "bobby78",
			"password": "pass",
		},
	},
	{
		"method": "GET",
		"path":   []string{"users", "alice1990"},
		"customer": map[string]interface{}{
			"login":    "alice1990",
			"password": "pass",
		},
	},
}

var store = inmem.NewFromReader(bytes.NewBufferString(`{
		"users": [
			{
				"login": "bobby78",
				"password": "pass"
			},
			{
				"login": "alice1990",
				"password": "pass"
			}
		]
	}`))

func PartialEval() {
	defer timeTrack(time.Now(), "partial evaluation took:")

	r := rego.New(
		rego.Query("data.example.allow"),
		rego.Module("example.rego", module),
		rego.Store(store),
	)

	pr, err := r.PartialResult(ctx)
	if err != nil {
		err = fmt.Errorf("partially evaluating Rego object : %v", err)
	}

	for i := range input {
		r = pr.Rego(
			rego.Input(input[i]),
			rego.Unknowns([]string{"data.users"}),
		)

		rs, err := r.Eval(ctx)
		if err != nil {
			err = fmt.Errorf("evaluating Rego object: %v", err)
		} else {
			fmt.Printf("passed: %v\n", rs[0].Expressions[0].Value)
		}
	}
}

func FullEval() {
	defer timeTrack(time.Now(), "full evaluation took:")

	r := rego.New(
		rego.Query("data.example.allow"),
		rego.Module("example.rego", module),
		rego.Store(store),
	)

	for i := range input {
		pq, err := r.PrepareForEval(ctx)
		if err != nil {
			err = fmt.Errorf("parse input arguments in preparation of evaluating them: %v", err)
		}

		rs, err := pq.Eval(ctx, rego.EvalInput(input[i]))
		if err != nil {
			err = fmt.Errorf("evaluating Rego object: %v", err)
		}

		fmt.Println("passed:", rs[0].Expressions[0])
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start).Nanoseconds()
	log.Printf("%s took %s", name, strconv.FormatInt(elapsed, 10))
}
