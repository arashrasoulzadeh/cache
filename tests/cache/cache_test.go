package cache

import (
	"cacher/pkg"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWrap(t *testing.T) {
	c := pkg.NewCache(true)

	value := "test"
	loops := 100000

	r := c.Wrap(context.Background(), "test", func() interface{} {
		return value
	})
	for range loops {
		go c.Wrap(context.Background(), "test", func() interface{} {
			return value
		})
	}

	time.Sleep(time.Second * 4)

	fmt.Println(c.Statistics(context.Background()))
	fmt.Println(c.AverageHitLatency(context.Background()))

	// Check the result of Wrap
	if r != "test" {
		t.Errorf("want test, got %v", r)
	}
}

//
//func TestWrapType(t *testing.T) {
//	c := pkg.NewCache()
//
//	// Use the WrapType generic method
//	f := func(data ...interface{}) string {
//		return "test-generic"
//	}
//
//	r := pkg.WrapType(context.Background(), "test-generic", c, f)
//
//	// Check the result of WrapType
//	if r != "test-generic" {
//		t.Errorf("want test-generic, got %v", r)
//	}
//}
