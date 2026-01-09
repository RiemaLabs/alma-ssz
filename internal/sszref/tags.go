package sszref

import (
	"reflect"
	"strconv"
	"strings"
)

type tagContext struct {
	sizes     []int
	maxes     []int
	isBitlist bool
}

func parseTagContext(tag reflect.StructTag) tagContext {
	return tagContext{
		sizes:     parseTagList(tag.Get("ssz-size")),
		maxes:     parseTagList(tag.Get("ssz-max")),
		isBitlist: tag.Get("ssz") == "bitlist",
	}
}

func parseTagList(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "?" {
			out = append(out, -1)
			continue
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			out = append(out, -1)
			continue
		}
		out = append(out, val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (ctx tagContext) size() (int, bool) {
	if len(ctx.sizes) == 0 {
		return -1, false
	}
	if ctx.sizes[0] < 0 {
		return -1, false
	}
	return ctx.sizes[0], true
}

func (ctx tagContext) max() (int, bool) {
	if len(ctx.maxes) == 0 {
		return -1, false
	}
	if ctx.maxes[0] < 0 {
		return -1, false
	}
	return ctx.maxes[0], true
}

func (ctx tagContext) shift() tagContext {
	next := tagContext{
		isBitlist: false,
	}
	if len(ctx.sizes) > 1 {
		next.sizes = ctx.sizes[1:]
	}
	if len(ctx.maxes) > 1 {
		next.maxes = ctx.maxes[1:]
	}
	return next
}
