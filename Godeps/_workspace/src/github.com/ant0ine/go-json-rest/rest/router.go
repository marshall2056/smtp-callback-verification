package rest

import (
	"errors"
	"github.com/ant0ine/go-json-rest/rest/trie"
	"net/url"
	"strings"
)

// TODO
// support for #param placeholder ?

type router struct {
	routes                 []*Route
	disableTrieCompression bool
	index                  map[*Route]int
	trie                   *trie.Trie
}

func escapedPath(urlObj *url.URL) string {
	// the escape method of url.URL should be public
	// that would avoid this split.
	parts := strings.SplitN(urlObj.RequestURI(), "?", 2)
	return parts[0]
}

// This validates the Routes and prepares the Trie data structure.
// It must be called once the Routes are defined and before trying to find Routes.
// The order matters, if multiple Routes match, the first defined will be used.
func (rt *router) start() error {

	rt.trie = trie.New()
	rt.index = map[*Route]int{}

	for i, route := range rt.routes {

		// PathExp validation
		if route.PathExp == "" {
			return errors.New("empty PathExp")
		}
		if route.PathExp[0] != '/' {
			return errors.New("PathExp must start with /")
		}
		urlObj, err := url.Parse(route.PathExp)
		if err != nil {
			return err
		}

		// work with the PathExp urlencoded.
		pathExp := escapedPath(urlObj)

		// make an exception for '*' used by the *splat notation
		// (at the trie insert only)
		pathExp = strings.Replace(pathExp, "%2A", "*", -1)

		// insert in the Trie
		err = rt.trie.AddRoute(
			strings.ToUpper(route.HttpMethod), // work with the HttpMethod in uppercase
			pathExp,
			route,
		)
		if err != nil {
			return err
		}

		// index
		rt.index[route] = i
	}

	if rt.disableTrieCompression == false {
		rt.trie.Compress()
	}

	return nil
}

// return the result that has the route defined the earliest
func (rt *router) ofFirstDefinedRoute(matches []*trie.Match) *trie.Match {
	minIndex := -1
	matchesByIndex := map[int]*trie.Match{}

	for _, result := range matches {
		route := result.Route.(*Route)
		routeIndex := rt.index[route]
		matchesByIndex[routeIndex] = result
		if minIndex == -1 || routeIndex < minIndex {
			minIndex = routeIndex
		}
	}

	return matchesByIndex[minIndex]
}

// Return the first matching Route and the corresponding parameters for a given URL object.
func (rt *router) findRouteFromURL(httpMethod string, urlObj *url.URL) (*Route, map[string]string, bool) {

	// lookup the routes in the Trie
	matches, pathMatched := rt.trie.FindRoutesAndPathMatched(
		strings.ToUpper(httpMethod), // work with the httpMethod in uppercase
		escapedPath(urlObj),         // work with the path urlencoded
	)

	// short cuts
	if len(matches) == 0 {
		// no route found
		return nil, nil, pathMatched
	}

	if len(matches) == 1 {
		// one route found
		return matches[0].Route.(*Route), matches[0].Params, pathMatched
	}

	// multiple routes found, pick the first defined
	result := rt.ofFirstDefinedRoute(matches)
	return result.Route.(*Route), result.Params, pathMatched
}

// Parse the url string (complete or just the path) and return the first matching Route and the corresponding parameters.
func (rt *router) findRoute(httpMethod, urlStr string) (*Route, map[string]string, bool, error) {

	// parse the url
	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return nil, nil, false, err
	}

	route, params, pathMatched := rt.findRouteFromURL(httpMethod, urlObj)
	return route, params, pathMatched, nil
}
