package engine

import (
	"sort"

	"matching-service/pkg/model"
)

// BasketMap 按篮子单号保存活跃内存篮子。
type BasketMap map[string]*Basket

// NeedIndex 按“还差金额”查找候选篮子。
// 实现可以使用前缀树、跳表、红黑树或分桶索引。
type NeedIndex interface {
	// Add 把篮子加入索引。
	Add(basket *Basket)

	// Remove 从索引中删除篮子。
	Remove(basket *Basket)

	// Update 在篮子还差金额变化后移动索引。
	Update(oldNeed int64, basket *Basket)

	// FindCompletable 返回可被这笔入金直接凑满的篮子。
	FindCompletable(amount int64, limit int) []*Basket

	// FindAcceptable 返回可以接收这笔入金但不会被直接凑满的篮子。
	FindAcceptable(amount int64, limit int) []*Basket
}

// BucketNeedIndex 是基于“还差金额分桶”的 NeedIndex 实现。
type BucketNeedIndex struct {
	needs   []int64                      // needs 是有序的还差金额列表
	buckets map[int64]map[string]*Basket // buckets 按还差金额保存篮子
	lookup  map[string]int64             // lookup 记录篮子当前所在的还差金额
}

// NewBucketNeedIndex 创建还差金额分桶索引。
func NewBucketNeedIndex() *BucketNeedIndex {
	return &BucketNeedIndex{
		buckets: make(map[int64]map[string]*Basket),
		lookup:  make(map[string]int64),
	}
}

// Add 把篮子加入索引。
func (idx *BucketNeedIndex) Add(basket *Basket) {
	if basket == nil || basket.Status != model.StatusWaiting {
		return
	}
	need := basket.NeedAmount()
	if idx.buckets[need] == nil {
		idx.buckets[need] = make(map[string]*Basket)
		idx.insertNeed(need)
	}
	idx.buckets[need][basket.BasketNo] = basket
	idx.lookup[basket.BasketNo] = need
}

// Remove 从索引中删除篮子。
func (idx *BucketNeedIndex) Remove(basket *Basket) {
	if basket == nil {
		return
	}
	need, ok := idx.lookup[basket.BasketNo]
	if !ok {
		need = basket.NeedAmount()
	}
	idx.removeFromNeed(need, basket.BasketNo)
}

// Update 在篮子还差金额变化后移动索引。
func (idx *BucketNeedIndex) Update(oldNeed int64, basket *Basket) {
	if basket == nil {
		return
	}
	idx.removeFromNeed(oldNeed, basket.BasketNo)
	idx.Add(basket)
}

// FindCompletable 返回可被这笔入金刚好凑满的篮子。
func (idx *BucketNeedIndex) FindCompletable(amount int64, limit int) []*Basket {
	out := make([]*Basket, 0, limit)
	bucket := idx.buckets[amount]
	for _, basket := range bucket {
		if basket.CanComplete(amount) {
			out = append(out, basket)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

// FindAcceptable 返回可以接收这笔入金但不会被直接凑满的篮子。
func (idx *BucketNeedIndex) FindAcceptable(amount int64, limit int) []*Basket {
	out := make([]*Basket, 0, limit)
	for i := sort.Search(len(idx.needs), func(i int) bool { return idx.needs[i] > amount }); i < len(idx.needs); i++ {
		for _, basket := range idx.buckets[idx.needs[i]] {
			if basket.CanAccept(amount) {
				out = append(out, basket)
				if len(out) >= limit {
					return out
				}
			}
		}
	}
	return out
}

// insertNeed 把还差金额插入有序列表。
func (idx *BucketNeedIndex) insertNeed(need int64) {
	pos := sort.Search(len(idx.needs), func(i int) bool { return idx.needs[i] >= need })
	if pos < len(idx.needs) && idx.needs[pos] == need {
		return
	}
	idx.needs = append(idx.needs, 0)
	copy(idx.needs[pos+1:], idx.needs[pos:])
	idx.needs[pos] = need
}

// removeFromNeed 从指定还差金额桶里删除篮子。
func (idx *BucketNeedIndex) removeFromNeed(need int64, basketNo string) {
	bucket := idx.buckets[need]
	if bucket == nil {
		delete(idx.lookup, basketNo)
		return
	}
	delete(bucket, basketNo)
	delete(idx.lookup, basketNo)
	if len(bucket) > 0 {
		return
	}
	delete(idx.buckets, need)
	pos := sort.Search(len(idx.needs), func(i int) bool { return idx.needs[i] >= need })
	if pos < len(idx.needs) && idx.needs[pos] == need {
		idx.needs = append(idx.needs[:pos], idx.needs[pos+1:]...)
	}
}
