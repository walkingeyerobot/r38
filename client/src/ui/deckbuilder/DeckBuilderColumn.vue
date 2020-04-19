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
          };
          e.dataTransfer.setData("text/plain", JSON.stringify(cardMove));
          e.dataTransfer.effectAllowed = "move";
        }
      },

      dragOver(e: DragEvent) {
        e.preventDefault();
        if (e.dataTransfer) {
          e.dataTransfer.dropEffect = "move";
          this.$el.classList.add("columnDrop");
        }
      },

      dragEnd(e: DragEvent) {
        e.preventDefault();
        if (e.dataTransfer) {
          this.$el.classList.remove("columnDrop");
        }
      },

      drop(e: DragEvent) {
        e.preventDefault();
        if (e.dataTransfer) {
          const cardMove: CardMove = JSON.parse(e.dataTransfer.getData("text/plain"));
          // TODO find correct index based on event target
          this.$tstore.commit("moveCard",
              {
                ...cardMove,
                targetMaindeck: this.maindeck,
                targetColumnIndex: this.columnIndex,
              });
        }
        this.$el.classList.remove("columnDrop");
      }
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
</style>