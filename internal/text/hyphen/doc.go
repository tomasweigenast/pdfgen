// Package hyphen finds hyphenation points in words using TeX pattern
// dictionaries.
//
// Dictionaries use the hyph_*.dic format shared by Hunspell and LibreOffice:
// an encoding line followed by Liang patterns, optionally with non-standard
// replacements such as "ab/c=d,1,2". A set of dictionaries is embedded and
// selected by language tag.
package hyphen
