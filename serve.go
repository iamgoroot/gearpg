package gearpg

import (
	"github.com/go-pg/pg/v11"
	"github.com/go-pg/pg/v11/orm"
	"github.com/go-pg/pg/v11/types"
	"github.com/gogearbox/gearbox"
	rqp "github.com/timsolov/rest-query-parser"
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
		m.Gear.Post(opt.Route, m.post(opt))
		m.Gear.Get(opt.Route, m.get(opt))
		m.Gear.Delete(opt.Route, m.delete(opt))
		m.Gear.Patch(opt.Route, m.patch(opt))
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
					q.OrderExpr("? DESC", types.Ident(sort.By))
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
