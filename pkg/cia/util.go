package cia

import "strconv"

func EndpointURL(collection string) string {
	return EndpointCollection() + collection
}

func PageURL(collection string, page int) string {
	if page < 1 {
		return EndpointCollection() + collection
	}
	return EndpointCollection() + collection + "?page=" + strconv.Itoa(page)
}
