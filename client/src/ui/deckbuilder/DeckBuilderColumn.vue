<template>
  <div
    class="_column"
    :class="{
      columnDrop: dropTargetIndex !== null,
      horizontal,
    }"
    @dragover="dragOver"
    @dragleave="dragLeave"
    @dragend="dragEnd"
    @drop="drop"
  >
    <div
      v-for="(card, index) in column"
      :key="card.id"
      @mousedown="preventMouseDown"
      @dragstart="dragStart($event, index)"
      @click.left="select(index)"
      @click.middle="zoom(index)"
      @dblclick="switchSection(index)"
      class="card"
      :class="{
        cardDropAbove: dropTargetIndex !== null && index === 0 && dropTargetIndex === 0,
        cardDropBelow: dropTargetIndex !== null && index === dropTargetIndex - 1,
        noSelectionRectangle: selectionRectangle === null,
        inSelectionRectangle: inSelectionRectangle.includes(index),
        inSelection: inSelection(index),
      }"
    >
      <img class="card-img" :src="getImageSrc(card.definition)" :alt="card.definition.name" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { MtgCard } from "@/draft/DraftState";
import {
  type CardColumn,
  type CardLocation,
  type CardMove,
  deckBuilderStore as store,
} from "@/state/DeckBuilderModule";
import { type Rectangle, intersects } from "@/util/rectangle";
import DeckBuilderSection from "./DeckBuilderSection.vue";

export default defineComponent({
  name: "DeckBuilderColumn",

  props: {
    column: {
      type: Array as () => CardColumn,
    },
    columnIndex: {
      type: Number,
      required: true,
    },
    deckIndex: {
      type: Number,
      required: true,
    },
    maindeck: {
      type: Boolean,
    },
    horizontal: {
      type: Boolean,
    },
    selectionRectangle: {
      type: Object as () => Rectangle | null,
    },
  },

  data: () => ({
    dropTargetIndex: null as number | null,
  }),

  computed: {
    selection(): CardLocation[] {
      return store.selection;
    },
    inSelectionRectangle(): number[] {
      const result = [];
      if (this.selectionRectangle) {
        for (let i = 0; i < this.$el.childElementCount; i++) {
          const child = <HTMLElement>this.$el.children[i];
          const childRect = {
            start: {
              x: child.offsetLeft,
              y: child.offsetTop,
            },
            end: {
              x: child.offsetLeft + child.offsetWidth,
              y: child.offsetTop + child.offsetHeight,
            },
          };
          if (intersects(childRect, this.selectionRectangle)) {
            result.push(i);
          }
        }
      }
      return result;
    },
  },

  methods: {
    getImageSrc(card: MtgCard): string {
      return card.image_uris[0];
    },

    dragStart(e: DragEvent, index: number) {
      let cardMove: CardMove;
      if (e.dataTransfer) {
        if (this.selection.length > 1 && this.inSelection(index)) {
          const dragImage = (<InstanceType<typeof DeckBuilderSection>>(
            this.$parent
          )).createDragImage();
          e.dataTransfer.setDragImage(dragImage, (<HTMLElement>this.$el).offsetWidth / 2, 20);
          cardMove = {
            deckIndex: this.deckIndex,
            source: this.selection,
            target: {
              columnIndex: 0,
              cardIndex: 0,
              maindeck: false,
            },
          };
        } else {
          store.selectCards([]);
          cardMove = {
            deckIndex: this.deckIndex,
            source: [
              {
                columnIndex: this.columnIndex,
                cardIndex: index,
                maindeck: this.maindeck,
              },
            ],
            target: {
              columnIndex: 0,
              cardIndex: 0,
              maindeck: false,
            },
          };
        }
        e.dataTransfer.setData("text/plain", JSON.stringify(cardMove));
        e.dataTransfer.effectAllowed = "move";
      }
    },

    getTargetIndex: function (e: DragEvent) {
      let targetIndex = 0;
      for (let i = 0; i < this.$el.childElementCount; i++) {
        const child = this.$el.children[i];
        const isTarget = child === e.target || child === (<Element>e.target).parentNode;
        if (isTarget) {
          targetIndex = i + 1;
        }
      }
      return targetIndex;
    },

    dragOver(e: DragEvent) {
      e.preventDefault();
      if (e.dataTransfer) {
        e.dataTransfer.dropEffect = "move";
        this.dropTargetIndex = this.getTargetIndex(e);
      }
    },

    dragLeave(e: DragEvent) {
      e.preventDefault();
      this.dropTargetIndex = null;
    },

    dragEnd(e: DragEvent) {
      e.preventDefault();
      this.dropTargetIndex = null;
      const dragImage = document.getElementById("dragImage");
      if (dragImage) {
        dragImage.remove();
      }
    },

    drop(e: DragEvent) {
      e.preventDefault();
      if (e.dataTransfer) {
        const cardMove: CardMove = JSON.parse(e.dataTransfer.getData("text/plain"));
        store.moveCard({
          ...cardMove,
          target: {
            columnIndex: this.columnIndex,
            cardIndex: this.getTargetIndex(e),
            maindeck: this.maindeck,
          },
        });
      }
      this.dropTargetIndex = null;
    },

    select(cardIndex: number) {
      store.selectCards([
        {
          columnIndex: this.columnIndex,
          cardIndex,
          maindeck: this.maindeck,
        },
      ]);
    },

    zoom(cardIndex: number) {
      store.zoomCard({
        columnIndex: this.columnIndex,
        cardIndex,
        maindeck: this.maindeck,
      });
    },

    switchSection(cardIndex: number) {
      store.moveCard({
        deckIndex: this.deckIndex,
        source: [
          {
            columnIndex: this.columnIndex,
            cardIndex,
            maindeck: this.maindeck,
          },
        ],
        target: {
          columnIndex: this.columnIndex,
          cardIndex: -1,
          maindeck: !this.maindeck,
        },
      });
    },

    preventMouseDown(e: MouseEvent) {
      e.stopPropagation();
    },

    inSelection(index: number) {
      return this.selection.some(
        (location) =>
          location.maindeck === this.maindeck &&
          location.columnIndex === this.columnIndex &&
          location.cardIndex === index,
      );
    },
  },
});
</script>

<style scoped>
._column {
  padding: 10px 10px 250px;
  width: 204px;
  flex: 0 0 auto;
}

._column.horizontal {
  padding: 3px 3px 250px;
  width: 104px;
}

.columnDrop {
  background: #ddd;
}

.horizontal .columnDrop {
  background: #555;
}

.card {
  height: 30px;
  overflow-y: visible;
  position: relative;
}

.horizontal .card {
  height: 15px;
}

.card-img {
  width: 200px;
  height: 279px;
  border: 2px solid transparent;
  border-radius: 10px;
}

.horizontal .card-img {
  width: 100px;
  height: 139px;
  border-radius: 6px;
}

.noSelectionRectangle > .card-img:hover,
.inSelectionRectangle > .card-img {
  border-color: #bbd;
}

.inSelection > .card-img,
.inSelection > .card-img:hover {
  border-color: #66e;
}

.inSelectionRectangle::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 279px;
  border-radius: 10px;
  background-color: #bbd8;
}

.cardDropAbove:before {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 10px;
  background: #00f;
  border-radius: 2px;
  pointer-events: none;
}

.cardDropBelow:before {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 10px;
  background: #00f;
  border-radius: 2px;
  pointer-events: none;
}
</style>
