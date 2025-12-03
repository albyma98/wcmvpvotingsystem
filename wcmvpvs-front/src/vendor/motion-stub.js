import { defineComponent, h, mergeProps, onMounted, ref } from 'vue';

const applyVariant = (el, variant) => {
  if (!el || !variant) {
    return;
  }
  const { transition, ...styles } = variant;
  if (transition) {
    const duration = (transition.duration ?? transition.d) ?? 0.35;
    const easing = transition.easing ?? transition.ease ?? 'ease';
    el.style.transition = `all ${duration}s ${easing}`;
  }
  Object.assign(el.style, styles);
};

export const Motion = defineComponent({
  name: 'MotionStub',
  props: {
    tag: { type: String, default: 'div' },
    initial: { type: Object, default: () => ({}) },
    enter: { type: Object, default: () => ({}) },
    hover: { type: Object, default: () => ({}) },
    press: { type: Object, default: () => ({}) },
  },
  setup(props, { slots }) {
    const el = ref(null);

    const applyEnter = () => applyVariant(el.value, { ...props.enter });

    const handleHover = (active) => {
      if (!props.hover || !Object.keys(props.hover).length) {
        return;
      }
      applyVariant(el.value, active ? { ...props.enter, ...props.hover } : { ...props.enter });
    };

    const handlePress = (active) => {
      if (!props.press || !Object.keys(props.press).length) {
        return;
      }
      applyVariant(el.value, active ? { ...props.enter, ...props.press } : { ...props.enter });
    };

    onMounted(() => {
      applyVariant(el.value, props.initial);
      requestAnimationFrame(() => {
        applyEnter();
      });
    });

    return () =>
      h(
        props.tag || 'div',
        mergeProps(
          {
            ref: el,
            onMouseenter: () => handleHover(true),
            onMouseleave: () => handleHover(false),
            onPointerdown: () => handlePress(true),
            onPointerup: () => handlePress(false),
          },
        ),
        slots.default ? slots.default() : [],
      );
  },
});

export const MotionPlugin = {
  install(app) {
    app.component('Motion', Motion);
  },
};

export default Motion;
