<template>
  <div
      class="column"
      @dragover="dragOver"
      @dragleave="dragEnd"
      @dragend="dragEnd"
      @drop="drop"
  >
    <div
        v-for="(card, index) in column"
        class="card"
    >
      <img
          class="card-img"
          :src="getImageSrc(card)"
          :alt="card.name"
          :data-index="index"
          @dragstart="dragStart"
      />
    </div>
  </div>
</template>

<script lang="ts">
  import Vue from 'vue';
  import {MtgCard} from "../../draft/DraftState.js";
  import {CardColumn, CardMove} from "../../state/store.js";

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
      }
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

      dragStart(e: DragEvent) {
        if (e.dataTransfer) {
          const targetElement = <HTMLElement>e.target;
          const cardMove: CardMove = {
            deckIndex: this.deckIndex,
            sourceMaindeck: this.maindeck,
            sourceColumnIndex: this.columnIndex,
            sourceCardIndex: Number(targetElement.dataset["index"]),
            targetMaindeck: false,
            targetColumnIndex: 0,
            targetCardIndex: 0,
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
          this.$el.classList.add("columnDrop");
          if (this.$el.childElementCount > 0) {
            let targetIndex = this.getTargetIndex(e);
            for (let i = 0; i < this.$el.childElementCount; i++) {
              const child = this.$el.children[i];
              child.classList.toggle("cardDropAbove",
                  i === 0 && targetIndex === 0);
              child.classList.toggle("cardDropBelow",
                  i === targetIndex - 1);
            }
          }
        }
      },

      dragEnd(e: DragEvent) {
        e.preventDefault();
        this.clearClasses();
      },

      drop(e: DragEvent) {
        e.preventDefault();
        if (e.dataTransfer) {
          const cardMove: CardMove = JSON.parse(e.dataTransfer.getData("text/plain"));
          this.$tstore.commit("moveCard",
              {
                ...cardMove,
                targetMaindeck: this.maindeck,
                targetColumnIndex: this.columnIndex,
                targetCardIndex: this.getTargetIndex(e),
              });
        }
        this.clearClasses();
      },

      clearClasses() {
        this.$el.classList.remove("columnDrop");
        for (const child of this.$el.children) {
          child.classList.remove("cardDropAbove");
          child.classList.remove("cardDropBelow");
        }
      },
    },
  });
</script>

<style scoped>

  .column {
    padding: 10px;
    width: 204px;
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

  .card-img:hover {
    border-color: #bbd;
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