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

var options = pg.Options{
	User:     "test",
	Password: "test",
	Database: "test",
	Addr:     "db:5432",
}

type MaTableModel struct {
	ID          int    `pg:"id0" json:"id0"`
	SomeKey     string `pg:"some_key0" json:"someKey0"`
	Description string `pg:"description0" json:"description0"`
}

type MaTableModel2 struct {
	ID2          int    `pg:"id1" json:"id1"`
	SomeKey2     string `pg:"some_key1" json:"someKey1"`
	Description2 string `pg:"description1" json:"description1"`
}

//func TestGeaRPG_Run(t *testing.T) {
//	background, app := prepare()
//	defer app.PG.Close(background)
//	t.Log(app.Gear.Start(":1234"))
//}
func TestGeaRPG_With(t *testing.T) {
	background, app := prepare()
	defer app.PG.Close(background)
	go func() {
		_ = app.Gear.Start(":1234")
	}()

	for i := 0; i < 2; i++ {
		tst := crudTest{url: fmt.Sprintf("http://127.0.0.1:1234/crud-name%d", i), i: i}
		tst.delete(fmt.Sprintf("?id%d[gt]=0", i))

		beforeCreate, err := tst.read("")
		assert.Equal(t, 0, beforeCreate)
		assert.Nil(t, err)

		err = tst.create()
		assert.Nil(t, err)
		t.Log("created entity")

		afterCreate, err := tst.read("")
		assert.Equal(t, 4, afterCreate)
		assert.Nil(t, err)

		beforeUpdate, err := tst.read(fmt.Sprintf("?description%[1]d[like]=find*&id%[1]d[eq]=1", i))
		assert.Equal(t, 0, beforeUpdate)
		assert.Nil(t, err)

		err = tst.update(fmt.Sprintf("?id%d[eq]=1", i))
		t.Log("updated entity")
		assert.Nil(t, err)

		afterUpdate, err := tst.read(fmt.Sprintf("?description%[1]d[like]=*_updated&id%[1]d[eq]=1", i))
		assert.Equal(t, 1, afterUpdate)
		assert.Nil(t, err)

		err = tst.delete(fmt.Sprintf("?id%d[eq]=1", i))
		assert.Nil(t, err)

		t.Log("deleted entity")

		read, err := tst.read("")
		assert.Nil(t, err)
		assert.Equal(t, 3, read)

		err = tst.delete(fmt.Sprintf("?id%d[gte]=1", i))
		assert.Nil(t, err)

		afterDelete, err := tst.read("")
		assert.Equal(t, 0, afterDelete)
		assert.Nil(t, err)

	}
}

func prepare() (context.Context, *GeaRPG) {
	background := context.Background()

	app := &GeaRPG{Gear: gearbox.New(), PG: pg.Connect(&options)}

	app.With(
		&Endpoint{
			Route: "/crud-name0",
			Validations: rqp.Validations{
				"sort": rqp.In(
					"someKey0",
					"description0",
				),
				"someKey0":     nil,
				"description0": nil,
				"id0":          nil,
			},
			Replacer: rqp.Replacer{
				"someKey0":     "some_key0",
				"description0": "description0",
			},
			MakeOne: func() interface{} {
				return &MaTableModel{}
			},
			MakeSlice: func() interface{} {
				return &[]MaTableModel{}
			},
		},
		&Endpoint{
			Route: "/crud-name1",
			Validations: rqp.Validations{
				"sort": rqp.In(
					"someKey1",
					"description1",
				),
				"someKey1":     nil,
				"description1": nil,
				"id1":          nil,
			},
			Replacer: rqp.Replacer{
				"someKey1":     "some_key1",
				"description1": "description1",
			},
			MakeOne: func() interface{} {
				return &MaTableModel2{}
			},
			MakeSlice: func() interface{} {
				return &[]MaTableModel2{}
			},
		},
	)
	_ = app.PG.Model((*MaTableModel)(nil)).CreateTable(background, nil)
	_ = app.PG.Model((*MaTableModel2)(nil)).CreateTable(background, nil)
	return background, app
}

type crudTest struct {
	url string
	i   int
}

func (c crudTest) update(q string) error {
	data, err := json.Marshal(c.mockModel(1, "updated"))
	req, err := http.NewRequest(http.MethodPatch, c.url+q, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	all, err := io.ReadAll(do.Body)
	fmt.Println(string(all), err)
	return err
}

func (c crudTest) read(q string) (int, error) {
	get, err := http.Get(c.url + q)
	if err != nil {
		return 0, err
	}
	data, err := ioutil.ReadAll(get.Body)
	if err != nil {
		return 0, err
	}
	var res []MaTableModel
	err = json.Unmarshal(data, &res)
	return len(res), err
}

func (c crudTest) delete(q string) error {
	req, err := http.NewRequest(http.MethodDelete, c.url+q, nil)
	if err != nil {
		return err
	}
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	all, err := io.ReadAll(do.Body)
	fmt.Println(string(all), err)
	return err
}

func (c crudTest) create() error {
	for i := 1; i < 5; i++ {
		data, err := json.Marshal(c.mockModel(i, "new"))
		if err != nil {
			return err
		}
		_, err = http.Post(c.url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c crudTest) mockModel(id int, add string) (model interface{}) {
	model = MaTableModel{
		ID:          id,
		SomeKey:     fmt.Sprintf("key[%d_%s]_%s", c.i, letters[:id%len(letters)], add),
		Description: fmt.Sprintf("descriptions %d_%s_%s", c.i, letters[:id%len(letters)], add),
	}
	if c.i%2 == 1 {
		model = MaTableModel2{
			ID2:          id,
			SomeKey2:     fmt.Sprintf("key[%d_%s]_%s", c.i, letters[:id%len(letters)], add),
			Description2: fmt.Sprintf("descriptions %d_%s_%s", c.i, letters[:id%len(letters)], add),
		}
	}
	return model
}

var letters = "abcdefghijklmnopqrstuvwxyz"
