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
        class="card"
        :class="{
            cardDropAbove: dropTargetIndex !== null && index === 0 && dropTargetIndex === 0,
            cardDropBelow: dropTargetIndex !== null && index === dropTargetIndex - 1,
            noSelection: selectionRectangle === null,
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
import { CardColumn, CardMove } from "../../state/DeckBuilderModule";
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
      if (e.dataTransfer) {
        const cardMove: CardMove = {
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
      }
      this.dropTargetIndex = null;
    },

    preventMouseDown(e: MouseEvent) {
      e.stopPropagation();
    },

    inSelection(index: number) {
      if (this.selectionRectangle) {
        const child = <HTMLElement>this.$el.children[index];
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
        return intersects(childRect, this.selectionRectangle);
      } else {
        return false;
      }
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

.noSelection > .card-img:hover, .inSelection > .card-img {
  border-color: #bbd;
}

.inSelection::after {
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