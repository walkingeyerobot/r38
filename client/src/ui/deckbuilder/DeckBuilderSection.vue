<template>
  <div class="_deck-builder-section">
    <div class="column-cnt"
         @mousedown="mouseDown"
         @mousemove="mouseMove"
         @mouseup="mouseUp"
         >
      <DeckBuilderColumn
          v-for="(column, index) in columns"
          ref="columns"
          :key="index"
          :column="column"
          :deckIndex="deckIndex"
          :maindeck="maindeck"
          :columnIndex="index"
          :selectionRectangle="selectionRectangle"
          />
      <div
          class="selection"
          :hidden="selectionRectangle === null"
          :style="{
          top: selectionTop,
          left: selectionLeft,
          width: selectionWidth,
          height: selectionHeight,
        }"></div>
    </div>
  </div>
</template>

<script lang="ts">
import Vue, { VueConstructor } from 'vue'
import DeckBuilderColumn from "./DeckBuilderColumn.vue";
import { CardColumn, CardLocation } from '../../state/DeckBuilderModule';
import { Point, Rectangle } from "../../util/rectangle";

export default (Vue as VueConstructor<Vue & {
  $refs: {
    columns: InstanceType<typeof DeckBuilderColumn>[],
  }
}>).extend({
  name: 'DeckBuilderSection',

  data: () => ({
    selectionRectangle: null as (Rectangle | null),
  }),

  computed: {
    selectionLeft(): string | null {
      if (this.selectionRectangle) {
        return Math.min(this.selectionRectangle.start.x, this.selectionRectangle.end.x) + "px";
      } else {
        return null;
      }
    },
    selectionTop(): string | null {
      if (this.selectionRectangle) {
        return Math.min(this.selectionRectangle.start.y, this.selectionRectangle.end.y) + "px";
      } else {
        return null;
      }
    },
    selectionWidth(): string | null {
      if (this.selectionRectangle) {
        return Math.abs(this.selectionRectangle.start.x - this.selectionRectangle.end.x) + "px";
      } else {
        return null;
      }
    },
    selectionHeight(): string | null {
      if (this.selectionRectangle) {
        return Math.abs(this.selectionRectangle.start.y - this.selectionRectangle.end.y) + "px";
      } else {
        return null;
      }
    },
  },

  components: {
    DeckBuilderColumn
  },

  props: {
    columns: {
      type: Array as () => CardColumn[]
    },
    deckIndex: {
      type: Number
    },
    maindeck: {
      type: Boolean
    }
  },

  methods: {

    relativePoint(clientX: number, clientY: number): Point {
      const rect = this.$el.getBoundingClientRect();
      return {
        x: clientX - rect.left,
        y: clientY - rect.top,
      }
    },

    mouseDown(e: MouseEvent) {
      const point = this.relativePoint(e.clientX, e.clientY);
      this.selectionRectangle = {
        start: {x: point.x, y: point.y},
        end: {x: point.x, y: point.y},
      };
      e.preventDefault();
    },

    mouseMove(e: MouseEvent) {
      if (this.selectionRectangle) {
        const point = this.relativePoint(e.clientX, e.clientY);
        this.selectionRectangle.end.x = point.x;
        this.selectionRectangle.end.y = point.y;
        e.preventDefault();
      }
    },

    mouseUp(e: MouseEvent) {
      if (this.selectionRectangle) {
        const selection: CardLocation[] =
            (this.$refs.columns)
                .flatMap((column, columnIndex) => column.inSelectionRectangle
                    .map(cardIndex => ({
                      columnIndex,
                      cardIndex,
                      maindeck: this.maindeck,
                    })));
        this.$tstore.commit("deckbuilder/selectCards", selection);
        this.selectionRectangle = null;
        e.preventDefault();
      }
    },
  },

});
</script>

<style scoped>
._deck-builder-section {
  overflow: scroll;
}

.column-cnt {
  display: flex;
  flex-direction: row;
  position: relative;
}

.selection {
  position: absolute;
  background: #b9f4c655;
  border: 2px solid #b9f4c6;
  box-sizing: border-box;
}

</style>
