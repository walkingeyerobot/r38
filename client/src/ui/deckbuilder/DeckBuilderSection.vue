<template>
  <div class="_deck-builder-section">
    <div class="column-cnt"
         ref="columnContent"
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
          :horizontal="horizontal"
          />
      <DeckBuilderColumn
          :column="[]"
          :deckIndex="deckIndex"
          :maindeck="maindeck"
          :columnIndex="columns.length"
          :selectionRectangle="selectionRectangle"
          :horizontal="horizontal"
          />
      <div
          class="selection"
          :hidden="selectionRectangle === null"
          :style="{
          top: selectionTop || undefined,
          left: selectionLeft || undefined,
          width: selectionWidth || undefined,
          height: selectionHeight || undefined,
        }"></div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue'
import DeckBuilderColumn from './DeckBuilderColumn.vue';
import { CardColumn, CardLocation, deckBuilderStore as store } from '../../state/DeckBuilderModule';
import { Point, Rectangle } from '../../util/rectangle';

export default defineComponent({
  name: 'DeckBuilderSection',

  components: {
    DeckBuilderColumn,
  },

  props: {
    columns: {
      type: Array as () => CardColumn[],
      required: true,
    },
    deckIndex: {
      type: Number,
      required: true,
    },
    maindeck: {
      type: Boolean,
      required: true,
    },
    horizontal: {
      type: Boolean,
      required: true,
    },
  },

  setup() {
    const columnElems = ref<Array<InstanceType<typeof DeckBuilderColumn>>>([]);
    const columnContentElem = ref<HTMLElement>(STANDIN_ELEM);

    return {
      columnElems,
      columnContentElem,
    };
  },

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
    selection(): CardLocation[] {
      return store.selection;
    },
  },

  methods: {

    relativePoint(clientX: number, clientY: number): Point {
      const rect = this.columnContentElem.getBoundingClientRect();
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
            (this.columnElems)
                .flatMap((column, columnIndex) => column.inSelectionRectangle
                    .map((cardIndex: number) => ({
                      columnIndex,
                      cardIndex,
                      maindeck: this.maindeck,
                    })));
        store.selectCards(selection);
        this.selectionRectangle = null;
        e.preventDefault();
      }
    },

    createDragImage() {
      const dragImage = <HTMLElement>this.columnContentElem.cloneNode(false);
      for (let columnIndex = 0; columnIndex < this.columnElems.length; columnIndex++) {
        if (this.selection.some(location =>
            location.maindeck === this.maindeck && location.columnIndex === columnIndex)) {
          const column = <HTMLElement>this.columnElems[columnIndex].$el.cloneNode(true);
          for (let cardIndex = column.childElementCount - 1; cardIndex >= 0; cardIndex--) {
            if (!this.selection.some(location =>
                location.maindeck === this.maindeck && location.columnIndex === columnIndex
                && location.cardIndex === cardIndex)) {
              column.removeChild(column.children[cardIndex]);
            }
          }
          dragImage.appendChild(column);
        }
      }
      dragImage.style.position = "absolute";
      dragImage.style.top = "-1000px";
      dragImage.id = "dragImage";
      document.body.appendChild(dragImage);
      return dragImage;
    }
  },

});

const STANDIN_ELEM = {} as any;
</script>

<style scoped>
._deck-builder-section {
  overflow-x: scroll;
  flex-grow: 1;
}

.column-cnt {
  display: flex;
  flex-direction: row;
  position: relative;
  padding-top: 20px;
}

.selection {
  position: absolute;
  background: #b9f4c655;
  border: 2px solid #b9f4c6;
  box-sizing: border-box;
}

</style>
