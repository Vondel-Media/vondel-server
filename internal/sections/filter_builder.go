package sections

import "github.com/Vondel-Media/vondel-server/internal/catalog"

type FilterBuilder = catalog.QueryBuilder

func NewFilterBuilder(alias string) *catalog.QueryBuilder {
	return catalog.NewQueryBuilder(alias)
}
