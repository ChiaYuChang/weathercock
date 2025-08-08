package utils

import (
	"github.com/pgvector/pgvector-go"
)

func ToFloat32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}

func ToPgVector(f32 []float32) pgvector.Vector {
	return pgvector.NewVector(f32)
}

func Ptr[T any](v T) *T {
	return &v
}

type option[T any] struct {
	isValid bool
	Value   T
}

func (o option[T]) IsValid() bool {
	return o.isValid
}

func (o option[T]) Get() T {
	if !o.isValid {
		panic("option is not valid")
	}
	return o.Value
}

func (o option[T]) OrElse(defaultVal T) T {
	if o.isValid {
		return o.Value
	}
	return defaultVal
}

func Option[T any](v T) option[T] {
	return option[T]{isValid: true, Value: v}
}
