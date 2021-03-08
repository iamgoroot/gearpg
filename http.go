package gearpg

import (
	"fmt"
	"github.com/gogearbox/gearbox"
	"net/http"
)

func (m GeaRPG) patch(opt *Endpoint) func(g gearbox.Context) {
	return func(g gearbox.Context) {
		if model := opt.MakeOne(); safe(g, g.ParseBody(model)) {
			if query, err := prepareQuery(m.PG, g, model, opt); safe(g, err) {
				if res, err := query.Update(g.Context()); safe(g, err) {
					safe(g, g.SendJSON(mutation{res.RowsAffected()}))
				}
			}
		}
	}
}

func (m GeaRPG) delete(opt *Endpoint) func(g gearbox.Context) {
	return func(g gearbox.Context) {
		if query, err := prepareQuery(m.PG, g, opt.MakeOne(), opt); safe(g, err) {
			if res, err := query.Delete(g.Context()); safe(g, err) {
				safe(g, g.SendJSON(mutation{res.RowsAffected()}))
			}
		}
	}
}

func (m GeaRPG) get(opt *Endpoint) func(g gearbox.Context) {
	return func(g gearbox.Context) {
		fmt.Println(opt)
		models := opt.MakeSlice()
		query, err := prepareQuery(m.PG, g, models, opt)
		_ = safe(g, err) && safe(g, query.Select(g.Context())) && safe(g, g.SendJSON(models))
	}
}

func (m GeaRPG) post(opt *Endpoint) func(g gearbox.Context) {
	return func(g gearbox.Context) {
		if model := opt.MakeOne(); safe(g, g.ParseBody(model)) {
			if _, err := m.PG.Model(model).Insert(g.Context()); safe(g, err) {
				safe(g, g.Status(http.StatusCreated).SendJSON(model))
			}
		}
	}
}
