package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
	"log"
	"strconv"
	"strings"
	"time"
)

var ctx = context.Background()

//go:embed example.rego
var module string

var input = map[string]interface{}{
	"method": "GET",
	"path":   []string{"users", "bob"},
	"customer": map[string]interface{}{
		"login":    "bob",
		"password": "pass",
	},
}

var store = inmem.NewFromReader(bytes.NewBufferString(`{
		"users": [
			{
				"login": "bob",
				"password": "pass"
			},
			{
				"login": "ali",
				"password": "pass"
			}
		]
	}`))

func PartialEval() {
	defer timeTrack(time.Now(), "partial evaluation took:")

	r := rego.New(
		rego.Query("data.example.allow == true"),
		rego.Module("example.rego", module),
		rego.Input(input),
		rego.Store(store),
		rego.Unknowns([]string{"data.users"}),
	)

	pq, err := r.Partial(ctx)
	if err != nil {
		log.Fatal(err)
	}

	conditions := make([]string, len(pq.Queries))
	for i := range pq.Queries {
		condition := toSQL(pq.Queries[i])
		conditions[i] = fmt.Sprintf("%s", condition)
		fmt.Println(toSQLWhere(pq.Queries[i]))
	}
	stmt := strings.Join(conditions, " OR ")
	fmt.Println(stmt)
}

func FullEval() {
	defer timeTrack(time.Now(), "full evaluation took:")

	r := rego.New(
		rego.Query("data.example.allow"),
		rego.Module("example.rego", module),
		rego.Store(store),
	)

	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		err = fmt.Errorf("parse input arguments in preparation of evaluating them: %v", err)
	}

	rs, err := pq.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		err = fmt.Errorf("evaluating Rego object: %v", err)
	}

	fmt.Println("passed:", rs.Allowed())
}

func toSQL(in ast.Body) string {
	var result []string
	for i := range in {
		expr := in[i]
		if !expr.IsCall() {
			continue
		}
		var op string
		switch v := expr.Operator(); v.String() {
		case "eq":
			op = " = "
		default:
			log.Fatalf("unsupported operator: %s", v)
		}

		l, r := expr.Operand(0).String(), expr.Operand(1).String()
		if strings.Contains(l, "data.users[_]") {
			l, r = strings.ReplaceAll(l, "data.users[_].", ""), r
		} else {
			l, r = strings.ReplaceAll(r, "data.users[_].", ""), l
		}
		q := strings.Join([]string{l, r}, op)
		result = append(result, q)
	}

	return strings.Join(result, " AND ")
}

func toSQLWhere(in ast.Body) map[string]string {
	result := make(map[string]string)
	for i := range in {
		expr := in[i]
		if !expr.IsCall() {
			continue
		}

		l, r := expr.Operand(0).String(), expr.Operand(1).String()
		if strings.Contains(l, "data.users[_]") {
			l, r = strings.ReplaceAll(l, "data.users[_].", ""), r
		} else {
			l, r = strings.ReplaceAll(r, "data.users[_].", ""), l
		}
		result[l] = r
	}
	return result
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start).Nanoseconds()
	log.Printf("%s took %s", name, strconv.FormatInt(elapsed, 10))
}
