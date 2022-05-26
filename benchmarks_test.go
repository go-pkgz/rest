package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmark_Stats(t *testing.T) {
	bench := NewBenchmarks()
	bench.update(time.Millisecond * 50)
	bench.update(time.Millisecond * 150)
	bench.update(time.Millisecond * 250)
	bench.update(time.Millisecond * 100)

	{
		res := bench.Stats(time.Minute)
		t.Logf("%+v", res)
		assert.Equal(t, BenchmarkStats{Requests: 4, RequestsSec: 4, AverageRespTime: 137,
			MinRespTime: (time.Millisecond * 50).Milliseconds(), MaxRespTime: (time.Millisecond * 250).Milliseconds()}, res)
	}

	{
		res := bench.Stats(time.Second * 5)
		t.Logf("%+v", res)
		assert.Equal(t, BenchmarkStats{Requests: 4, RequestsSec: 4, AverageRespTime: 137,
			MinRespTime: (time.Millisecond * 50).Milliseconds(), MaxRespTime: (time.Millisecond * 250).Milliseconds()}, res)
	}

	{
		res := bench.Stats(time.Millisecond * 999)
		t.Logf("%+v", res)
		assert.Equal(t, BenchmarkStats{}, res)
	}
}

func TestBenchmark_Stats2s(t *testing.T) {
	bench := NewBenchmarks()
	bench.update(time.Millisecond * 50)
	bench.update(time.Millisecond * 150)
	bench.update(time.Millisecond * 250)
	time.Sleep(time.Second)
	bench.update(time.Millisecond * 100)

	res := bench.Stats(time.Minute)
	t.Logf("%+v", res)
	assert.Equal(t, BenchmarkStats{Requests: 4, RequestsSec: 2, AverageRespTime: 137,
		MinRespTime: (time.Millisecond * 50).Milliseconds(), MaxRespTime: (time.Millisecond * 250).Milliseconds()}, res)
}

func TestBenchmark_Cleanup(t *testing.T) {
	bench := NewBenchmarks()
	for i := 0; i < 1000; i++ {
		bench.nowFn = func() time.Time {
			return time.Date(2022, 5, 15, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second) // every 2s fake time
		}
		bench.update(time.Millisecond * 50)
	}

	{
		res := bench.Stats(time.Hour)
		t.Logf("%+v", res)
		assert.Equal(t, BenchmarkStats{Requests: 900, RequestsSec: 1, AverageRespTime: 50,
			MinRespTime: (time.Millisecond * 50).Milliseconds(), MaxRespTime: (time.Millisecond * 50).Milliseconds()}, res)
	}
	{
		res := bench.Stats(time.Minute)
		t.Logf("%+v", res)
		assert.Equal(t, BenchmarkStats{Requests: 60, RequestsSec: 1, AverageRespTime: 50,
			MinRespTime: (time.Millisecond * 50).Milliseconds(), MaxRespTime: (time.Millisecond * 50).Milliseconds()}, res)
	}

	assert.Equal(t, 900, bench.data.Len())
}

func TestBenchmarks_Handler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		time.Sleep(time.Millisecond * 50)
		require.NoError(t, err)
	})

	bench := NewBenchmarks()
	ts := httptest.NewServer(bench.Handler(handler))
	defer ts.Close()

	for i := 0; i < 100; i++ {
		resp, err := ts.Client().Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{
		res := bench.Stats(time.Minute)
		t.Logf("%+v", res)
		assert.Equal(t, 100, res.Requests)
		assert.True(t, res.RequestsSec <= 20 && res.RequestsSec >= 10)
		assert.InDelta(t, 50, res.AverageRespTime, 10)
		assert.InDelta(t, 50, res.MinRespTime, 10)
		assert.InDelta(t, 50, res.MaxRespTime, 10)
		assert.True(t, res.MaxRespTime >= res.MinRespTime)
	}

	{
		res := bench.Stats(time.Minute * 15)
		t.Logf("%+v", res)
		assert.Equal(t, 100, res.Requests)
		assert.True(t, res.RequestsSec <= 20 && res.RequestsSec >= 10)
		assert.InDelta(t, 50, res.AverageRespTime, 10)
		assert.InDelta(t, 50, res.MinRespTime, 10)
		assert.InDelta(t, 50, res.MaxRespTime, 10)
		assert.True(t, res.MaxRespTime >= res.MinRespTime)
	}
}
