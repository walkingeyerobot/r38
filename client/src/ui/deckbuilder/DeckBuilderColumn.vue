<template>
  <div
      class="_column"
      :class="{columnDrop: dropTargetIndex !== null}"
      @dragover="dragOver"
      @dragleave="dragEnd"
      @dragend="dragEnd"
      @drop="drop"
      >
    <div
        v-for="(card, index) in column"
        :key="card.id"
        @mousedown="preventMouseDown"
        @dragstart="dragStart($event, index)"
        @click="select(index)"
        class="card"
        :class="{
            cardDropAbove: dropTargetIndex !== null && index === 0 && dropTargetIndex === 0,
            cardDropBelow: dropTargetIndex !== null && index === dropTargetIndex - 1,
            noSelectionRectangle: selectionRectangle === null,
            inSelectionRectangle: inSelectionRectangle.includes(index),
            inSelection: inSelection(index),
        }"
        >
      <img
          class="card-img"
          :src="getImageSrc(card.definition)"
          :alt="card.definition.name"
          />
    </div>
  </div>
</template>

<script lang="ts">
import Vue from 'vue';
import { MtgCard } from "../../draft/DraftState.js";
import { CardColumn, CardLocation, CardMove } from "../../state/DeckBuilderModule";
import { intersects, Rectangle } from "../../util/rectangle";

export default Vue.extend({
  name: 'DeckBuilderColumn',

  props: {
    column: {
      type: Array as () => CardColumn
    },
    columnIndex: {
      type: Number
    },
    deckIndex: {
      type: Number
    },
    maindeck: {
      type: Boolean
    },
    selectionRectangle: {
      type: Object as () => (Rectangle | null)
    },
  },

  data: () => ({
    dropTargetIndex: null as (number | null),
  }),

  computed: {
    selection(): CardLocation[] {
      return this.$tstore.state.deckbuilder.selection;
    },
    inSelectionRectangle(): number[] {
      const result = [];
      if (this.selectionRectangle) {
        for (let i = 0 ;i < this.$el.childElementCount; i++) {
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
      if (process.env.NODE_ENV == 'development') {
        return `http://api.scryfall.com/cards/${card.set}/`
            + `${card.collector_number}?format=image&version=normal`;
      } else {
        return `/proxy/${card.set}/${card.collector_number}`;
      }
    },

    dragStart(e: DragEvent, index: number) {
      let cardMove: CardMove;
      if (e.dataTransfer) {
        if (this.selection.length > 1
            && this.inSelection(index)) {
          const dragImage = <HTMLElement>this.$el.cloneNode(true);
          for (let i = dragImage.childElementCount - 1; i >= 0; i--) {
            if (!this.inSelection(i)) {
              dragImage.removeChild(dragImage.children[i]);
            }
          }
          dragImage.style.position = "absolute";
          dragImage.style.top = "-1000px";
          dragImage.id = "dragImage";
          document.body.appendChild(dragImage);
          e.dataTransfer.setDragImage(dragImage, (<HTMLElement>this.$el).offsetWidth / 2, 20);
          // TODO expand to selection
          cardMove = {
            deckIndex: this.deckIndex,
            source: {
              columnIndex: this.columnIndex,
              cardIndex: index,
              maindeck: this.maindeck,
            },
            target: {
              columnIndex: 0,
              cardIndex: 0,
              maindeck: false,
            },
          };
        } else {
          this.$tstore.commit("deckbuilder/selectCards", []);
          cardMove = {
            deckIndex: this.deckIndex,
            source: {
              columnIndex: this.columnIndex,
              cardIndex: index,
              maindeck: this.maindeck,
            },
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

    dragEnd(e: DragEvent) {
      e.preventDefault();
      this.dropTargetIndex = null;
    },

    drop(e: DragEvent) {
      e.preventDefault();
      if (e.dataTransfer) {
        const cardMove: CardMove = JSON.parse(e.dataTransfer.getData("text/plain"));
        this.$tstore.commit("deckbuilder/moveCard",
            {
              ...cardMove,
              target: {
                columnIndex: this.columnIndex,
                cardIndex: this.getTargetIndex(e),
                maindeck: this.maindeck,
              },
            });
        const dragImage = document.getElementById("dragImage");
        if (dragImage) {
          dragImage.remove();
        }
      }
      this.dropTargetIndex = null;
    },

    select(cardIndex: number) {
      this.$tstore.commit("deckbuilder/selectCards", [{
        columnIndex: this.columnIndex,
        cardIndex,
        maindeck: this.maindeck,
      }]);
    },

    preventMouseDown(e: MouseEvent) {
      e.stopPropagation();
    },

    inSelection(index: number) {
      return this.selection.some(location =>
          location.maindeck === this.maindeck
          && location.columnIndex === this.columnIndex
          && location.cardIndex === index);
    }
  },
});
</script>

<style scoped>

._column {
  padding: 10px;
  width: 204px;
  flex: 0 0 auto;
  padding-bottom: 250px;
}

.columnDrop {
  background: #ddd;
}

.card {
  height: 30px;
  overflow-y: visible;
  position: relative;
}

.card-img {
  width: 200px;
  height: 279px;
  border: 2px solid transparent;
  border-radius: 10px;
}

.noSelectionRectangle > .card-img:hover, .inSelectionRectangle > .card-img {
  border-color: #bbd;
}

.inSelection > .card-img, .inSelection > .card-img:hover {
  border-color: #66e;
}

.inSelectionRectangle::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 279px;
  border-radius: 10px;
  background-color: #bbd8;
}

.cardDropAbove:before {
  content: '';
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
  content: '';
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