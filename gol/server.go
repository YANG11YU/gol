package gol

import (
	"sync"
)

type ServerStruct struct{}

type Request struct {
	Main      *MainStruct
	Turn       int
	StartIndex int
	EndIndex   int
}

type Response struct {
	RpcStatus int
	Main     *MainStruct
	Turn      int
}


func (this *ServerStruct) Caculate(req Request, resp *Response) error {
	//fmt.Println(req.Turn, req.StartIndex, req.EndIndex)
	for i := 0; i < req.Turn; i++ {
		req.Main.Calcaulate(req.StartIndex, req.EndIndex)
	}

	resp.Main = req.Main
	resp.RpcStatus = 0
	resp.Turn = req.Turn
	return nil
}

// 下一轮所有的细胞状态计算, 并发逻辑在这里实现
func (w *MainStruct) Calcaulate(startIndex, endIndex int) {
	wg := sync.WaitGroup{}
	//每个worker需要计算的宽度
	lenght := (endIndex - startIndex + 1) / w.T
	for i := 0; i < w.T; i++ {
		wg.Add(1)
		// 启动worker进行计算
		start := i * lenght + startIndex
		end := start + lenght - 1
		if i == w.T-1 {
			end = endIndex
		}
		go func(wg *sync.WaitGroup, index, startIndex, endIndex int) {
			defer wg.Done()
			for x := 0; x < w.H; x++ {
				for y := startIndex; y <= endIndex; y++ {
					w.TmpMesh.Set(x, y, w.NowMesh.NextCalculate(x, y))
				}
			}
		}(&wg, i, start, end)
	}
	wg.Wait()

	w.NowMesh, w.TmpMesh = w.TmpMesh, w.NowMesh
}