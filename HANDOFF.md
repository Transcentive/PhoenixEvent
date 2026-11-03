# Handoff: open questions and work in hand

Read this at the start of a session. Remove an item only once the owner has
answered it and the answer is recorded in the specification.

## Work in hand (November 1, 2026)

- **Phase 2 exit, built and not committed.** `backend/internal/server/isolation_test.go`
  tests that one organization's token, sign-in cookie and email links are
  refused at another organization's address. It found that a logout sent to
  one organization's address could end another organization's session; logout
  now ends only a session of the organization at its address
  (`accounts.Logout` takes the organization). All backend tests pass. Waiting
  for the owner to test logging out and back in on the Mac, then his word to
  commit and push.
- **Phase 3, O3 Policies and S1 Create event.** Not started. Waiting for the
  owner's answers to the five questions below. S1 has no phone drawing yet: draw
  it and show it to the owner before building.

## Five questions before building O3 and S1 (asked November 1, 2026; open)

**1. Board first or chair first.** Asked plainly, after the first wording
was too convoluted:

> Every new event needs an event board and a chair. The spec allows two ways to
> set them up:
>
> - **Board first:** Organization Leadership picks the board members, and they
>   then choose their chair from among themselves.
> - **Chair first:** Organization Leadership picks the chair, and the chair then
>   picks the board members.
>
> The spec never says which way the first release uses.
>
> **Question:** when someone creates an event, which way should it go?
>
> - **(a)** The person creating the event chooses, on the Create event screen
>   (recommended).
> - **(b)** Always board first.
> - **(c)** Always chair first.

**2. What the Policies page (O3) shows in the first release.** Elaborated at
the owner's request:

> The Policies page (O3) is where an Organization Administrator sets rules for
> every event in the organization. The drawing has four sections:
>
> 1. **Registration:** a checkbox, "Approve each registration before the person
>    chooses stations" (when on, the event board approves each person who
>    registers before they can sign up for stations); and a choice of who takes
>    part: people who register (free), people who pay, or both.
> 2. **Waiting lists:** whether a full station keeps a waiting list.
> 3. **Event Administrator:** whether the person who runs the event's
>    day-to-day work takes one of the board's seats or works alongside the
>    board.
> 4. **Lowering a station's capacity:** what happens when the board cuts a
>    station's spots below the number of people already in it.
>
> The spec (16B.2) puts these in the first release: Registration, with people
> who register only (paying comes later); and the Event Administrator choice
> (Organization Leadership appoints that person when forming the board,
> feature 5). Waiting lists and lowering capacity are a later release (the
> first release always refuses lowering capacity).
>
> **Proposed now:** Registration with the approval checkbox, and "Who takes
> part" as a single line, "People register to take part", since there is
> nothing to choose yet; Event Administrator with its two choices, "works
> alongside" as the default. Waiting lists and lowering capacity left off the
> page until their releases.
>
> **Question:** is that what you want, or should the page show all four
> sections now, with the later two marked "comes in a later release"?

**3. Who can see the event on S1.** The first release has private events only
(16B.2, feature 4). Recommended: a single line, "Private. Members of
[organization] see it after logging in.", instead of three choices.

**4. After Create event.** S7 and S8 (forming the board) come with feature 5
and are not built. Recommended: go back to O1 with the new event listed, and
build S7 and S8 next.

**5. Required fields on S1.** Recommended: the event name and the board
capacity (at least 1) required; date, location and description optional
(5A.4, step 1).

## Other open items

- **Logging in as Ramruva for step 3.** The owner said testing step 3 required
  logging in as ramruva.kshatriaya; no earlier message said so. Asked where he
  saw it; no answer yet. Step 3 was tested and pushed (a6b64a3) since.
- **O8 shows roles set to Private**, for the time being, by the owner's ruling
  of November 1, 2026 (spec v1.08). Revisit after review.
