package runtime

import (
	"context"
	"fmt"

	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
)

var _ store.KVStoreService = (*GenesisKVStoreServie)(nil)

type genesisContextKeyType struct{}

var genesisContextKey = genesisContextKeyType{}

type genesisContext struct {
	state  store.WriterMap
	didRun bool
}

func makeGenesisContext(state store.WriterMap) genesisContext {
	return genesisContext{
		state: state,
	}
}

func (g *genesisContext) Run(
	ctx context.Context,
	fn func(ctx context.Context) error,
) (store.WriterMap, error) {
	ctx = context.WithValue(ctx, genesisContextKey, g)
	err := fn(ctx)
	if err != nil {
		return nil, err
	}
	g.didRun = true
	return g.state, nil
}

type GenesisKVStoreServie struct {
	actor            []byte
	executionService store.KVStoreService
}

func NewGenesisKVService(
	actor []byte,
	execution store.KVStoreService,
) *GenesisKVStoreServie {
	return &GenesisKVStoreServie{
		actor:            actor,
		executionService: execution,
	}
}

// OpenKVStore implements store.KVStoreService.
func (g *GenesisKVStoreServie) OpenKVStore(ctx context.Context) store.KVStore {
	v := ctx.Value(genesisContextKey)
	if v == nil {
		return g.executionService.OpenKVStore(ctx)
	}
	genCtx, ok := v.(*genesisContext)
	if !ok {
		panic(fmt.Errorf("unexpected genesis context type: %T", v))
	}
	if genCtx.didRun {
		return g.executionService.OpenKVStore(ctx)
	}
	state, err := genCtx.state.GetWriter(g.actor)
	if err != nil {
		panic(err)
	}
	return state
}

type GenesisHeaderService struct {
	executionService header.Service
}

// HeaderInfo implements header.Service.
func (g *GenesisHeaderService) HeaderInfo(ctx context.Context) header.Info {
	v := ctx.Value(genesisContextKey)
	if v == nil {
		return g.executionService.HeaderInfo(ctx)
	}
	genCtx, ok := v.(*genesisContext)
	if !ok {
		panic(fmt.Errorf("unexpected genesis context type: %T", v))
	}
	if genCtx.didRun {
		return g.executionService.HeaderInfo(ctx)
	}
	return header.Info{}
}

func NewGenesisHeaderService(executionService header.Service) *GenesisHeaderService {
	return &GenesisHeaderService{
		executionService: executionService,
	}
}

var _ header.Service = (*GenesisHeaderService)(nil)
