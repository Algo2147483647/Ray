package math_lib

const (
	EPS = 1e-5 // 微小量，用于浮点数比较
)

func DecomposeBlocks(rows, cols, eachBlockRows, eachBlockCols int64) [][2]int64 {
	if eachBlockRows <= 0 || eachBlockCols <= 0 {
		return nil
	}
	R := (rows + eachBlockRows - 1) / eachBlockRows
	C := (cols + eachBlockCols - 1) / eachBlockCols
	if R <= 0 || C <= 0 {
		return nil
	}
	minRC := R
	if C < R {
		minRC = C
	}
	maxLayer := (minRC - 1) / 2
	var result [][2]int64
	for l := maxLayer; l >= 0; l-- {
		top := l
		bottom := R - 1 - l
		left := l
		right := C - 1 - l
		for j := left; j <= right; j++ {
			result = append(result, [2]int64{top, j})
		}
		for i := top + 1; i <= bottom-1; i++ {
			result = append(result, [2]int64{i, right})
		}
		if top != bottom {
			for j := right; j >= left; j-- {
				result = append(result, [2]int64{bottom, j})
			}
		}
		if left != right {
			for i := bottom - 1; i >= top+1; i-- {
				result = append(result, [2]int64{i, left})
			}
		}
	}
	return result
}

func BlockRange(rows, cols, eachBlockRows, eachBlockCols, blockRow, blockCol int64) (rowSt, rowEd, colSt, colEd int64) {
	rowSt = blockRow * eachBlockRows
	rowEd = rowSt + eachBlockRows
	if rowEd > rows {
		rowEd = rows
	}
	colSt = blockCol * eachBlockCols
	colEd = colSt + eachBlockCols
	if colEd > cols {
		colEd = cols
	}
	return
}
