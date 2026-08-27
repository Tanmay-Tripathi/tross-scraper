// Package voyager talks to LinkedIn's private internal JSON API. Every wire-format
// detail is quarantined here, so an upstream change moves this package alone.
package voyager

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Envelope is the flat shape every Voyager response arrives in: Data refers to
// objects by ID, the objects themselves sit in Included.
type Envelope struct {
	Data     json.RawMessage   `json:"data"`
	Included []json.RawMessage `json:"included"`
}

// Entity is the minimum every Included object carries: its ID and type name.
type Entity struct {
	EntityUrn string `json:"entityUrn"`
	Type      string `json:"$type"`
}

// Graph is a decoded Voyager response with Included indexed for lookup.
type Graph struct {
	Data json.RawMessage
	// byUrn maps an entityUrn to its raw object.
	byUrn map[string]json.RawMessage
	// byType groups urns by $type, so a caller can ask for "every Position".
	byType map[string][]string
	order  []string
}

// NewGraph decodes a Voyager response and indexes it.
func NewGraph(payload []byte) (*Graph, error) {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode voyager envelope: %w", err)
	}

	graph := &Graph{
		Data:   envelope.Data,
		byUrn:  make(map[string]json.RawMessage, len(envelope.Included)),
		byType: make(map[string][]string),
		order:  make([]string, 0, len(envelope.Included)),
	}

	for _, raw := range envelope.Included {
		var entity Entity
		if err := json.Unmarshal(raw, &entity); err != nil {
			// One malformed entity must not sink the profile.
			continue
		}
		if entity.EntityUrn == "" {
			continue
		}

		graph.byUrn[entity.EntityUrn] = raw
		graph.order = append(graph.order, entity.EntityUrn)
		if entity.Type != "" {
			graph.byType[entity.Type] = append(graph.byType[entity.Type], entity.EntityUrn)
		}
	}

	return graph, nil
}

// Size reports how many entities were indexed; 0 means an empty or blocked response.
func (g *Graph) Size() int { return len(g.byUrn) }

// Types counts the distinct $type values present — how we discover a profile's contents.
func (g *Graph) Types() map[string]int {
	counts := make(map[string]int, len(g.byType))
	for typeName, urns := range g.byType {
		counts[typeName] = len(urns)
	}
	return counts
}

// Resolve decodes the entity at urn into dest.
func (g *Graph) Resolve(urn string, dest any) bool {
	raw, ok := g.byUrn[urn]
	if !ok {
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

// ResolveAll decodes every urn it can, skipping missing or malformed ones.
func ResolveAll[T any](g *Graph, urns []string) []T {
	results := make([]T, 0, len(urns))
	for _, urn := range urns {
		var item T
		if g.Resolve(urn, &item) {
			results = append(results, item)
		}
	}
	return results
}

// Collection follows a section pointer to its CollectionResponse and decodes the
// entities it lists. dash puts exactly one collection between a profile and every
// section, so this is the hop that replaces legacy's flat "*elements" lists.
func Collection[T any](g *Graph, pointer string) []T {
	if pointer == "" {
		return nil
	}

	var ref collectionRef
	if !g.Resolve(pointer, &ref) {
		return nil
	}

	return ResolveAll[T](g, ref.Elements)
}

// ByType returns every entity whose $type ends with typeSuffix, in LinkedIn's
// order. Suffix matching survives LinkedIn renaming its versioned namespace.
func ByType[T any](g *Graph, typeSuffix string) []T {
	var results []T
	for _, urn := range g.order {
		raw := g.byUrn[urn]
		var entity Entity
		if err := json.Unmarshal(raw, &entity); err != nil {
			continue
		}
		if !strings.HasSuffix(entity.Type, typeSuffix) {
			continue
		}
		var item T
		if json.Unmarshal(raw, &item) == nil {
			results = append(results, item)
		}
	}
	return results
}
