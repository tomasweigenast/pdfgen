// Package boxes defines the CSS box tree: the formatting structure of CSS 2.1
// chapter 9 and the page boxes of CSS Paged Media 3.
//
// Every box embeds [BoxFields], which holds the style, the used values set
// during layout and the flags shared by all boxes, and boxes with children
// embed [ParentFields]. Boxes are pointers and layout mutates them in place;
// [Box.Copy] makes a shallow copy and [Box.DeepCopy] also copies the
// descendants.
//
// The outside and inside behaviours that CSS gives to boxes are interfaces
// with unexported marker methods: [BlockLevel], [InlineLevel],
// [BlockContainer], [AtomicInline], [Replaced], [FlexContainer],
// [GridContainer] and the table roles. Callers use type switches or
// assertions on those interfaces instead of the concrete types.
package boxes
