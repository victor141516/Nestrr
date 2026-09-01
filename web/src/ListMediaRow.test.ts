import { fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import { session } from "./api";
import ListMediaRow from "./ListMediaRow.vue";

const item = {
  mediaType: "movie" as const,
  tmdbId: 550,
  title: "El club de la lucha",
  overview: "Un oficinista insomne conoce a Tyler Durden.",
  addedAt: "2025-03-14T09:26:53Z",
};
const lists = [
  {
    id: 1,
    systemKey: "favorites" as const,
    name: "Favorites",
    color: "coral" as const,
    icon: "heart",
    itemCount: 1,
    createdAt: "",
    updatedAt: "",
  },
  {
    id: 4,
    name: "Cine",
    color: "violet" as const,
    icon: "list",
    itemCount: 0,
    createdAt: "",
    updatedAt: "",
  },
];

afterEach(() => {
  session.user = null;
});

describe("list media row", () => {
  it("shows the snapshot and emits the selected list action", async () => {
    session.user = {
      id: 1,
      username: "admin",
      isAdmin: true,
      language: "es-ES",
      accentColor: "#ff624f",
    };
    const view = render(ListMediaRow, {
      props: {
        item,
        lists,
        memberships: [
          {
            listId: 1,
            systemKey: "favorites",
            mediaType: "movie",
            tmdbId: 550,
            addedAt: item.addedAt,
          },
        ],
        pending: true,
      },
      global: {
        stubs: {
          RouterLink: { template: "<a><slot /></a>" },
        },
      },
    });

    expect(view.getByText(item.overview)).toBeTruthy();
    expect(view.container.querySelector(".pending-removal")).toBeTruthy();
    const favorite = view.getByRole("button", { name: "Favoritos" });
    expect(favorite.getAttribute("aria-pressed")).toBe("true");
    await fireEvent.click(favorite);
    expect(view.emitted()["toggle-list"]?.[0]).toEqual([lists[0]]);
  });
});
