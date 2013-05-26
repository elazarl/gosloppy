/*
	Package scopes allows you to traverse through go/ast nodes, and know the scope of each
	node you're visiting.

	Usage:

	Implement scopes.Visitor interface, and use scopes.Lookup to search for elements in current scope.

*/
package scopes
