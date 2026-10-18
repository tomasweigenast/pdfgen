package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestPageBackgroundGradients(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`linear_gradients_1`,
			`
			_____
			_____
			_____
			BBBBB
			BBBBB
			RRRRR
			RRRRR
			RRRRR
			RRRRR
			`,
			`
			<style>@page { size: 5px 9px; background: linear-gradient(
			      white, white 3px, blue 0, blue 5px, red 0, red
			    )
			`,
		},
		{
			`linear_gradients_10`,
			`
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			`,
			`
			<style>@page { size: 9px 5px; background: linear-gradient(
			      45deg, blue 3px, red 3px, red 3px, blue 3px
			    )
			`,
		},
		{
			`linear_gradients_11`,
			`
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
			`,
			`
			<style>@page { size: 9px 5px; background: linear-gradient(
			      to right, blue 3px, red 3px, red 4px, blue 4px
			    )
			`,
		},
		{
			`linear_gradients_12`,
			`
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			`,
			`
			<style>@page { size: 9px 5px; background:
			      repeating-linear-gradient(to right, red 3px, blue 3px, blue 4px, red 4px
			    )
			`,
		},
		{
			`linear_gradients_13`,
			`
			_____
			_____
			_____
			SSSSS
			SSSSS
			RRRRR
			RRRRR
			RRRRR
			RRRRR
			`,
			`
			<style>@page { size: 5px 9px; background: linear-gradient(
			      white, white 3px, rgba(255, 0, 0, 0.751) 0, rgba(255, 0, 0, 0.751) 5px,
			      red 0, red
			    )
			`,
		},
		{
			`linear_gradients_2`,
			`
			_____
			_____
			_____
			BBBBB
			BBBBB
			RRRRR
			RRRRR
			RRRRR
			RRRRR
			`,
			`
			<style>@page { size: 5px 9px; background: linear-gradient(
			      white 3px, blue 0, blue 5px, red 0
			    )
			`,
		},
		{
			`linear_gradients_3`,
			`
			___BBrrrr
			___BBrrrr
			___BBrrrr
			___BBrrrr
			___BBrrrr
			`,
			`
			<style>@page { size: 9px 5px; background: linear-gradient(
			      to right, white 3px, blue 0, blue 5px, red 0
			    )
			`,
		},
		{
			`linear_gradients_4`,
			`
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
			`,
			`
			<style>@page { size: 10px 5px; background: linear-gradient(
			      to right, blue 5px, blue 6px, red 6px, red 9px
			    )
			`,
		},
		{
			`linear_gradients_5`,
			`
			rBrrzBrrrB
			rBrrrBrrrB
			rBrrrBrrrB
			rBrrrBrrrB
			rBrrzBrrrB
			`,
			`
			  <style>@page { size: 10px 5px; background: repeating-linear-gradient(
			  to right, blue 50%, blue 60%, red 60%, red 90%
			)
			`,
		},
		{
			`linear_gradients_6`,
			`
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
			`,
			`
			<style>@page { size: 9px 5px; background: linear-gradient(
			      to right, blue 3px, blue 3px, red 3px, red 3px
			    )
			`,
		},
		{
			`linear_gradients_7`,
			`
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
			`,
			`
			<style>@page { size: 9px 5px; background:
			    repeating-linear-gradient(
			      to right, black 3px, black 3px, #800080 3px, #800080 3px
			    )
			`,
		},
		{
			`linear_gradients_8`,
			`
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			`,
			`
			<style>@page { size: 9px 5px; background:
			      repeating-linear-gradient(to right, blue 3px)
			`,
		},
		{
			`linear_gradients_9`,
			`
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			`,
			`
			<style>@page { size: 9px 5px; background:
			      repeating-linear-gradient(45deg, blue 3px)
			`,
		},
		{
			`linear_gradients_currentcolor`,
			`
			KKKKK
			KKKKK
			KKKKK
			KKKKK
			KKKKK
			`,
			`
			<style>@page { size: 5px 5px; background: linear-gradient(currentcolor)
			`,
		},
		{
			`linear_gradients_hints`,
			`
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			zzzzz
			`,
			`
			<style>@page { size: 5px 9px; background: linear-gradient(
			      white, 8.9px, red
			    )
			`,
		},
		{
			`linear_gradients_hints_percentage`,
			`
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			zzzzz
			`,
			`
			<style>@page { size: 5px 9px; background: linear-gradient(
			      white, 99%, red
			    )
			`,
		},
		{
			`radial_gradients_1`,
			`
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
			`,
			`
			<style>@page { size: 6px; background:
			      radial-gradient(red -30%, blue -10%)
			`,
		},
		{
			`radial_gradients_2`,
			`
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
			`,
			`
			<style>@page { size: 6px; background:
			      radial-gradient(red 110%, blue 130%)
			`,
		},
		{
			`radial_gradients_3`,
			`
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
			`,
			`
			<style>@page { size: 10px 16px; background:
			      radial-gradient(red 20%, blue 80%)
			`,
		},
		{
			`radial_gradients_4`,
			`
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
			`,
			`
			<style>@page { size: 10px 16px; background:
			      radial-gradient(red 50%, blue 50%)
			`,
		},
		{
			`radial_gradients_5`,
			`
			SzzzzzzzzS
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			SzzzzzzzzS
			`,
			`
			<style>@page { size: 10px 16px; background:
			      radial-gradient(red 50%, rgba(255, 0, 0, 0.751) 50%)
			`,
		},
		{
			`radial_gradients_hints`,
			`
			zzzzzz
			zzBBzz
			zBBBBz
			zBBBBz
			zzBBzz
			zzzzzz
			`,
			`
			<style>@page { size: 6px; background:
			      radial-gradient(blue, 4px, white)
			`,
		},
		{
			`radial_gradients_hints_percentage`,
			`
			zzzzzz
			zzBBzz
			zBBBBz
			zBBBBz
			zzBBzz
			zzzzzz
			`,
			`
			<style>@page { size: 6px; background:
			      radial-gradient(blue, 99%, white)
			`,
		},
		{
			`radial_gradients_negative`,
			`
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzRRzzzB
			BzzzRRzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
			`,
			`
			<style>@page { size: 10px 16px; background:
			      radial-gradient(circle, red -1px, red 3px, blue 3px, blue 6px)
			`,
		},
		{
			`radial_gradients_repeating`,
			`
			RzzzzzzzzR
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zBzzRRzzBz
			zBzzRRzzBz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			RzzzzzzzzR
			`,
			`
			<style>@page { size: 10px 16px; background:
			      repeating-radial-gradient(circle, red 0, red 3px, blue 3px, blue 6px)
			`,
		},
		{
			`radial_gradients_repeating_negative`,
			`
			RzzzzzzzzR
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zBzzRRzzBz
			zBzzRRzzBz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			RzzzzzzzzR
			`,
			`
			<style>@page { size: 10px 16px; background:
			      repeating-radial-gradient(circle, blue -3px, blue 0, red 0, red 3px)
			`,
		},
		{
			`radial_gradients_repeating_outer`,
			`
			RzzzzzzzzR
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zBzzRRzzBz
			zBzzRRzzBz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			RzzzzzzzzR
			`,
			`
			<style>@page { size: 10px 16px; background:
			      repeating-radial-gradient(circle, red 6px, red 9px, blue 9px, blue 12px)
			`,
		},
		{
			`radial_gradients_repeating_outer_partial`,
			`
			RzzzzzzzzR
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zBzzRRzzBz
			zBzzRRzzBz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			RzzzzzzzzR
			`,
			`
			<style>@page { size: 10px 16px; background:
			      repeating-radial-gradient(circle, blue 3px, blue 6px, red 6px, red 9px)
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}
