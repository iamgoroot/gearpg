package gearpg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg/v11"
	"github.com/gogearbox/gearbox"
	"github.com/stretchr/testify/assert"
	rqp "github.com/timsolov/rest-query-parser"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

type MaTableModel struct {
	ID          int
	SomeKey     string `pg:"some_key"`
	Description string `pg:"description"`
}

func TestGeaRPG_With(t *testing.T) {
	background := context.Background()
	app := &GeaRPG{Gear: gearbox.New(), PG: pg.Connect(&pg.Options{
		User:     "test",
		Password: "test",
		Database: "test",
		Addr:     "db:5432",
	})}
	defer app.PG.Close(background)
	_ = app.PG.Model((*MaTableModel)(nil)).CreateTable(background, nil)

	app.With(
		&Endpoint{
			Route: "/crud-name",
			Validations: rqp.Validations{
				"sort": rqp.In(
					"someKey",
					"description",
				),
				"someKey":     nil,
				"description": nil,
				"id":          nil,
			},
			Replacer: rqp.Replacer{
				"someKey":     "some_key",
				"description": "description",
			},
			MakeOne: func() interface{} {
				return &MaTableModel{}
			},
			MakeSlice: func() interface{} {
				return &[]MaTableModel{}
			},
		},
	)

	go func() {
		_ = app.Gear.Start(":1234")
	}()
	beforeCreate := read("")
	assert.Equal(t, 0, beforeCreate)

	create()
	t.Log("created entity")

	afterCreate := read("")
	assert.Equal(t, 4, afterCreate)

	beforeUpdate := read("?description[like]=find*&id[eq]=1")
	assert.Equal(t, 0, beforeUpdate)

	update()
	t.Log("updated entity")

	afterUpdate := read("?description[like]=find*&id[eq]=1")
	assert.Equal(t, 1, afterUpdate)

	delete("?id[eq]=1")
	t.Log("deleted entity")

	assert.Equal(t, 3, read(""))
	delete("?id[gt]=1")
	assert.Equal(t, 0, read(""))

}

func update() {
	data, _ := json.Marshal(MaTableModel{
		ID:          666, //doesn't update pk
		SomeKey:     "updated key",
		Description: "find me",
	})
	req, err := http.NewRequest(http.MethodPatch, "http://test:1234/crud-name?id[eq]=1", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		panic(err)
	}
	do, err := http.DefaultClient.Do(req)
	all, err := io.ReadAll(do.Body)
	fmt.Println(string(all), err)

}

func read(q string) int {
	get, err := http.Get("http://test:1234/crud-name" + q)
	if err != nil {
		panic(err)
	}
	data, _ := ioutil.ReadAll(get.Body)
	var res []MaTableModel
	json.Unmarshal(data, &res)
	return len(res)
}

func delete(q string) {
	req, err := http.NewRequest(http.MethodDelete, "http://test:1234/crud-name"+q, nil)
	if err != nil {
		panic(err)
	}
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	all, err := io.ReadAll(do.Body)
	fmt.Println(string(all), err)
}

func create() {
	for i := 1; i < 5; i++ {
		data, _ := json.Marshal(MaTableModel{
			ID:          i,
			SomeKey:     fmt.Sprintf("key[%s]", letters[:i%len(letters)]),
			Description: fmt.Sprintf("descriptions %s", letters[:i%len(letters)]),
		})
		_, err := http.Post("http://test:1234/crud-name", "application/json", bytes.NewBuffer(data))
		if err != nil {
			panic(err)
		}
	}

}

var letters = "abcdefghijklmnopqrstuvwxyz"
