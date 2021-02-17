package gearpg

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/go-pg/pg/types"
	"github.com/gogearbox/gearbox"
	rqp "github.com/timsolov/rest-query-parser"
	"net/http"
)

type (
	GeaRPG struct {
		Gear gearbox.Gearbox
		PG   *pg.DB
	}
	Endpoint struct {
		//mount path
		Route string
		//rest-query-parser validations
		Validations rqp.Validations
		//rest-query-parser field to db column mapping
		Replacer rqp.Replacer
		//factory that makes one instance of request entity
		MakeOne func() interface{}
		//factory that makes slice request entity
		MakeSlice func() interface{}
	}
	mutation struct {
		AffectedRows int
	}
)

func (m GeaRPG) With(options ...*Endpoint) {
	for _, opt := range options {
		m.Gear.Post(opt.Route, func(g gearbox.Context) {
			if model := opt.MakeOne(); safe(g, g.ParseBody(model)) && safe(g, m.PG.Insert(model)) {
				safe(g, g.Status(http.StatusCreated).SendJSON(model))
			}
		})
		m.Gear.Get(opt.Route, func(g gearbox.Context) {
			models := opt.MakeSlice()
			query, err := prepareQuery(m.PG, g, models, opt)
			_ = safe(g, err) && safe(g, query.Select()) && safe(g, g.SendJSON(models))
		})
		m.Gear.Delete(opt.Route, func(g gearbox.Context) {
			if query, err := prepareQuery(m.PG, g, opt.MakeOne(), opt); safe(g, err) {
				if res, err := query.Delete(); safe(g, err) {
					safe(g, g.SendJSON(mutation{res.RowsAffected()}))
				}
			}
		})
		m.Gear.Patch(opt.Route, func(g gearbox.Context) {
			if model := opt.MakeOne(); safe(g, g.ParseBody(model)) {
				if query, err := prepareQuery(m.PG, g, model, opt); safe(g, err) {
					if res, err := query.Update(); safe(g, err) {
						safe(g, g.SendJSON(mutation{res.RowsAffected()}))
					}
				}
			}
		})
	}
}

func prepareQuery(db *pg.DB, g gearbox.Context, model interface{}, options *Endpoint) (q *orm.Query, err error) {
	url := string(g.Context().Request.RequestURI())
	query := rqp.New().SetValidations(options.Validations)
	if err = query.SetUrlString(url); err == nil {
		if err = query.Parse(); err == nil {
			query.ReplaceNames(options.Replacer)
			q = db.Model(model).Limit(query.Limit).Offset(query.Offset)
			if where := query.Where(); where != "" {
				q.Where(where, query.Args()...)
			}
			for _, sort := range query.Sorts {
				if sort.Desc {
					q.OrderExpr("? DESC", types.F(sort.By))
				} else {
					q.Order(sort.By)
				}
			}
		}
	}
	return
}

func safe(g gearbox.Context, err error) bool {
	return !(err != nil && g.Status(gearbox.StatusBadRequest).SendString(err.Error()).Context() != nil)
}
