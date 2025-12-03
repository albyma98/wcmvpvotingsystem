const applyStyles = (target, vars = {}) => {
  if (!target) return;
  const clone = { ...vars };
  delete clone.duration;
  delete clone.ease;
  delete clone.transformOrigin;
  delete clone.stagger;
  delete clone.repeat;
  delete clone.onComplete;
  if (clone.opacity !== undefined) {
    target.style.opacity = clone.opacity;
    delete clone.opacity;
  }
  Object.assign(target.style || target, clone);
};

const createTimeline = () => {
  const api = {
    to(target, vars = {}, position) {
      applyStyles(target, vars);
      if (typeof vars.onComplete === 'function') {
        vars.onComplete();
      }
      return api;
    },
    kill() {},
  };
  return api;
};

const to = (target, vars = {}) => {
  applyStyles(target, vars);
  if (typeof vars.onComplete === 'function') {
    vars.onComplete();
  }
  return { kill() {} };
};

const fromTo = (target, fromVars = {}, toVars = {}) => {
  applyStyles(target, fromVars);
  return to(target, toVars);
};

export const gsap = { timeline: createTimeline, to, fromTo };
export default gsap;
