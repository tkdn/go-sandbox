package main

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Rob Pike が 2012 waza にて講演した "Concurrency is not Prallelism" における
// Load Balancer イメージ実装をトレースしたもの。
// https://go.dev/talks/2012/waza.slide#45
// https://gist.github.com/rushilgupta/228dfdf379121cb9426d5e90d34c5b96#load-balancer-architecture
//
// Data Flow
// |Client1|         |Load|  <-DONE-- |Worker1| processing R3 coming from Client1
// |Client2| --REQ-> |Blncr| --WOK->  |Worker2| processing R8 coming from Client2
//                                    |Worker3| processing R5 coming from Client1
//                           <-RESP-- response of R4 to Client2
//
//	- クライアントは Request 構造体に data を詰めて req channel へ送信する
//	- LB は req channel で Request を待ち受ける
//	- LB は Worker を選び work channel に Request を送信する
//	- Worker は Request を受け取り, data を処理する(sin(data)など)
//	- Worker は done channel を使って LB を更新し、LB　は更新された情報で load balance する
//	- Worker は sin(data) を res channel へ書き込む(Request 構造体で返す)
//
// チャンネルは計4つ登場する。req, res, work, done

// reqClientCount はリクエストするクライアントの数を表す。
const reqClientCount = 100

// workerSize はリクエストを処理する Worker の数を表す。
const workerSize = 10

func main() {
	req := make(chan Request)
	for range reqClientCount {
		go createRequest(req)
	}
	newLoadBalancer().balance(req)
}

// Request は LB へ送られるリクエストを表現した構造体。
type Request struct {
	data int
	res  chan float64 // res はクライアントごとに持つレスポンスを待ち構えるチャンネル
}

// createRequest はクライアントの作成とリクエストをする。
// どのクライアントも無限ループするgoroutineで、
// ループ内ではLB(で待ち受けるチャンネル)へ送信するリクエスト(Request)を生成している。
// レスポンスについては、リクエストはクライアントごとに共通のチャンネルを使用する。
func createRequest(req chan Request) {
	res := make(chan float64)
	for {
		// ランダムにsleepを入れる
		time.Sleep(time.Duration(rand.Int63n(int64(time.Millisecond))))
		req <- Request{int(rand.Int31n(90)), res}
		// チャンネルからレスポンスを読み込む
		<-res
	}
}

// Worker は LB からのリクエスト処理を受け付ける構造体。
type Worker struct {
	idx     int          // ヒープインデックス
	work    chan Request // work チャンネル
	pending int          // このWorkerがどれだけリクエストを保留しているかの数
}

// do は無限ループする Worker goroutine を開始する。
// ループ内では Request 構造体(と構造体におけるdata計算)を待ち構え、done channel はブロックされる。
// Worker は複数のリクエストを受け付け、保留中のリクエストはリクエスト数を記録する。
func (w *Worker) do(done chan *Worker) {
	for {
		// work channel からリクエストを抽出する
		req := <-w.work
		// res channel へ計算結果を書き込む
		req.res <- math.Sin(float64(req.data))
		// done channel へ書き込む
		done <- w
	}
}

// Pool は Worker プールを表す。
type Pool []*Worker

// LoadBalancer は LB を表現した構造体。
// done チャンネルでは Worker からの書き込みを期待し、
// heap に通知することで pending カウンタを減少させる。
type LoadBalancer struct {
	pool Pool
	done chan *Worker
}

// newLoadBalancer は LB の初期化を行う。
// Workerプールの数、リクエスト数を読み込み、
// heap に規定の数だけ Worker を作成しプールする。
// 作成された Worker は goroutin を開始して処理待ち構える。
func newLoadBalancer() *LoadBalancer {
	done := make(chan *Worker, workerSize)
	b := &LoadBalancer{make(Pool, 0, workerSize), done}
	for range workerSize {
		w := &Worker{work: make(chan Request, reqClientCount)}
		heap.Push(&b.pool, w)
		go w.do(b.done)
	}
	return b
}

// balance は req channel で Request を待ち構え req を dispatch し、
// done channel で完了を待ち構える。バランスの結果を出力する。
func (b *LoadBalancer) balance(req chan Request) {
	for {
		select {
		case request := <-req:
			b.dispatch(request)
		case w := <-b.done:
			b.completed(w)
		}
		b.print()
	}
}

// dispatch は最も負荷の低い Worker へ処理を渡し heap を更新する。
func (b *LoadBalancer) dispatch(req Request) {
	w := heap.Pop(&b.pool).(*Worker)
	w.work <- req
	w.pending++
	heap.Push(&b.pool, w)
}

// completed は Worker の保留数を減退させ heap から削除し Pool に書き戻す。
func (b *LoadBalancer) completed(w *Worker) {
	w.pending--
	heap.Remove(&b.pool, w.idx)
	heap.Push(&b.pool, w)
}

// print はバランス結果を出力する。
func (b *LoadBalancer) print() {
	sum := 0
	sumsq := 0
	for _, w := range b.pool {
		fmt.Printf("%d ", w.pending)
		sum += w.pending
		sumsq += w.pending * w.pending
	}
	avg := float64(sum) / float64(len(b.pool))
	variance := float64(sumsq)/float64(len(b.pool)) - avg*avg
	fmt.Printf(" %.2f %.2f\n", avg, variance)
}

// 以下は heap.Interface を満たすための実装

var _ heap.Interface = (*Pool)(nil)

func (p Pool) Len() int { return len(p) }

func (p Pool) Less(i, j int) bool {
	return p[i].pending < p[j].pending
}

func (p *Pool) Swap(i, j int) {
	a := *p
	a[i], a[j] = a[j], a[i]
	a[i].idx = i
	a[j].idx = j
}

func (p *Pool) Push(x interface{}) {
	n := len(*p)
	w := x.(*Worker)
	w.idx = n
	*p = append(*p, w)
}

func (p *Pool) Pop() interface{} {
	old := *p
	n := len(old)
	w := old[n-1]
	w.idx = -1 // for safety
	*p = old[0 : n-1]
	return w
}
