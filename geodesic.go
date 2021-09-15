// package geodesic
//
// API for the geodesic routines in Go (cgo-enabled version)
//
// This an implementation in Go of the geodesic algorithms described in
// - C. F. F. Karney, Algorithms for geodesics,
//   J. Geodesy 87, 43--55 (2013);
//   DOI: 10.1007/s00190-012-0578-z;
//   addenda: https://geographiclib.sourceforge.io/geod-addenda.html;
//   link: https://doi.org/10.1007/s00190-012-0578-z;
//
// Copyright (c) Charles Karney (2012-2021) <charles@karney.com> and licensed
// under the MIT/X11 License.  For more information, see
// https://geographiclib.sourceforge.io/
//
// Ported to Go by Joshua Baker <joshbaker77@gmail.com> and licensed
// under the MIT License.
package geodesic

/*
#cgo CFLAGS: -O3
#cgo LDFLAGS: -lm
#include <stdbool.h>
#include "geodesic.h"
*/
import "C"

// WGS84 conforming ellispoid
// https://en.wikipedia.org/wiki/World_Geodetic_System
var WGS84 = NewEllipsoid(6378137.0, float64(1.)/float64(298.257223563))

// Ellipsoid is an object for performing geodesic operations.
type Ellipsoid struct {
	g C.struct_geod_geodesic
}

// NewEllipsoid initializes a new geodesic ellipsoid object.
// Param a is the equatorial radius (meters).
// Param f is the flattening.
func NewEllipsoid(radius, flattening float64) *Ellipsoid {
	e := new(Ellipsoid)
	C.geod_init(&e.g, C.double(radius), C.double(flattening))
	return e
}

// Inverse solve the inverse geodesic problem.
//
// Param lat1 is latitude of point 1 (degrees).
// Param lon1 is longitude of point 1 (degrees).
// Param lat2 is latitude of point 2 (degrees).
// Param lon2 is longitude of point 2 (degrees).
// Out param ps12 is a pointer to the distance from point 1 to point 2 (meters).
// Out param pazi1 is a pointer to the azimuth at point 1 (degrees).
// Out param pazi2 is a pointer to the (forward) azimuth at point 2 (degrees).
//
// lat1 and lat2 should be in the range [-90,+90].
// The values of azi1 and azi2 returned are in the range [-180,+180].
// Any of the "return" arguments, ps12, etc., may be replaced with nil, if you
// do not need some quantities computed.
//
// The solution to the inverse problem is found using Newton's method.  If
// this fails to converge (this is very unlikely in geodetic applications
// but does occur for very eccentric ellipsoids), then the bisection method
// is used to refine the solution.
func (e *Ellipsoid) Inverse(
	lat1, lon1, lat2, lon2 float64,
	s12, azi1, azi2 *float64,
) {
	C.geod_inverse(&e.g,
		C.double(lat1), C.double(lon1), C.double(lat2), C.double(lon2),
		(*C.double)(s12), (*C.double)(azi1), (*C.double)(azi2))
}

// Direct solves the direct geodesic problem.
//
// Param g is a pointer to the geod_geodesic object specifying the ellipsoid.
// Param lat1 is the latitude of point 1 (degrees).
// Param lon1 is the longitude of point 1 (degrees).
// Param azi1 is the azimuth at point 1 (degrees).
// Param s12 is the distance from point 1 to point 2 (meters). negative is ok.
// Out param plat2 is a pointer to the latitude of point 2 (degrees).
// Out param plon2 is a pointer to the longitude of point 2 (degrees).
// Out param pazi2 is a pointer to the (forward) azimuth at point 2 (degrees).
//
// lat1 should be in the range [-90,+90].
// The values of lon2 and azi2 returned are in the range [-180,+180].
// Any of the "return" arguments, plat2, etc., may be replaced with nil, if you
// do not need some quantities computed.
func (e *Ellipsoid) Direct(
	lat1, lon1, azi1, s12 float64,
	lat2, lon2, azi2 *float64,
) {
	C.geod_direct(&e.g,
		C.double(lat1), C.double(lon1), C.double(azi1), C.double(s12),
		(*C.double)(lat2), (*C.double)(lon2), (*C.double)(azi2))
}

// Polygon struct for accumulating information about a geodesic polygon.
// Used for computing the perimeter and area of a polygon.
// This must be initialized from Ellipsoid.PolygonInit before use.
type Polygon struct {
	e *Ellipsoid
	p C.struct_geod_polygon
}

// PolygonInit initializes a polygon.
// Param polyline for polyline instead of a polygon.
//
// If polyline is not set, then the sequence of vertices and edges added by
// Polygon.AddPoint() and Polygon.AddEdge() define a polygon and
// the perimeter and area are returned by Polygon.Compute().
// If polyline is set, then the vertices and edges define a polyline and
// only the perimeter is returned by Polygon.Compute().
//
// The area and perimeter are accumulated at two times the standard floating
// point precision to guard against the loss of accuracy with many-sided
// polygons.  At any point you can ask for the perimeter and area so far.
func (e *Ellipsoid) PolygonInit(polyline bool) Polygon {
	var p Polygon
	if polyline {
		C.geod_polygon_init(&p.p, 1)
	} else {
		C.geod_polygon_init(&p.p, 0)
	}
	p.e = e
	return p
}

// AddPoint adds a point to the polygon or polyline.
//
// Param lat is the latitude of the point (degrees).
// Param lon is the longitude of the point (degrees).
func (p *Polygon) AddPoint(lat, lon float64) {
	C.geod_polygon_addpoint(&p.e.g, &p.p, C.double(lat), C.double(lon))
}

// Compute the results for a polygon
//
// Param reverse, if set then clockwise (instead of
//   counter-clockwise) traversal counts as a positive area.
// Param sign, if set then return a signed result for the area if
//   the polygon is traversed in the "wrong" direction instead of returning
//   the area for the rest of the earth.
// Out param pA is a pointer to the area of the polygon (meters-squared);
// Out param pP is a pointer to the perimeter of the polygon or length of the
//   polyline (meters).
// Returns the number of points.
//
// The area and perimeter are accumulated at two times the standard floating
// point precision to guard against the loss of accuracy with many-sided
// polygons.  Arbitrarily complex polygons are allowed.  In the case of
// self-intersecting polygons the area is accumulated "algebraically", e.g.,
// the areas of the 2 loops in a figure-8 polygon will partially cancel.
// There's no need to "close" the polygon by repeating the first vertex.  Set
// pA or pP to nil, if you do not want the corresponding quantity returned.
//
// More points can be added to the polygon after this call.
func (p *Polygon) Compute(reverse, sign bool, area, perimeter *float64) int {
	var creverse, csign C.int
	if reverse {
		creverse = 1
	}
	if sign {
		csign = 1
	}
	return int(C.geod_polygon_compute(&p.e.g, &p.p, creverse, csign,
		(*C.double)(area), (*C.double)(perimeter)))
}

// AddEdge adds an edge to the polygon or polyline.
//
// Param azi is the azimuth at current point (degrees).
// Param s is the distance from current point to next point (meters).
func (p *Polygon) AddEdge(azi, s float64) {
	C.geod_polygon_addedge(&p.e.g, &p.p, C.double(azi), C.double(s))
}

// Clear the polygon, allowing a new polygon to be started.
func (p *Polygon) Clear() {
	C.geod_polygon_clear(&p.p)
}
