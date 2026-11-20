package pmdata

// parquetDoubles PMData Parquet 里 LIST<double>（list.element）结构。
type parquetDoubles struct {
	List []struct {
		Element float64 `parquet:"element"`
	} `parquet:"list"`
}

func (p parquetDoubles) values() []float64 {
	if len(p.List) == 0 {
		return nil
	}
	out := make([]float64, 0, len(p.List))
	for _, e := range p.List {
		out = append(out, e.Element)
	}
	return out
}