package cia

import "strconv"

func EndpointURL(collection string) string {
	return EndpointCollection() + collection
}

func PageURL(collection string, page int) string {
	return EndpointCollection() + collection + "?page=" + strconv.Itoa(page)
}
