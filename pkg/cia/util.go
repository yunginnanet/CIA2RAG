package cia

import "strconv"

func EndpointURL(collection string) string {
	return EndpointCollection() + collection
}

func PageURL(collection string, page int) string {
	if page <= 0 {
		page = 1
	}
	if page < 2 {
		return EndpointCollection() + collection
	}
	return EndpointCollection() + collection + "?page=" + strconv.Itoa(page)
}
