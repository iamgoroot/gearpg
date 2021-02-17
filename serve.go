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
		DB *pg.DB
		GB gearbox.Gearbox
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
		m.GB.Post(opt.Route, func(g gearbox.Context) {
			if model := opt.MakeOne(); safe(g, g.ParseBody(model)) && safe(g, m.DB.Insert(model)) {
				safe(g, g.Status(http.StatusCreated).SendJSON(model))
			}
		})
		m.GB.Get(opt.Route, func(g gearbox.Context) {
			models := opt.MakeSlice()
			query, err := prepareQuery(m.DB, g, models, opt)
			_ = safe(g, err) && safe(g, query.Select()) && safe(g, g.SendJSON(models))
		})
		m.GB.Delete(opt.Route, func(g gearbox.Context) {
			if query, err := prepareQuery(m.DB, g, opt.MakeOne(), opt); safe(g, err) {
				if res, err := query.Delete(); safe(g, err) {
					safe(g, g.SendJSON(mutation{res.RowsAffected()}))
				}
			}
		})
		m.GB.Patch(opt.Route, func(g gearbox.Context) {
			if model := opt.MakeOne(); safe(g, g.ParseBody(model)) {
				if query, err := prepareQuery(m.DB, g, model, opt); safe(g, err) {
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
	if safe(g, query.SetUrlString(url)) {
		query.ReplaceNames(options.Replacer)
		if safe(g, query.Parse()) {
			q = db.Model(model).Where(query.Where(), query.Args()...).Limit(query.Limit).Offset(query.Offset)
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
