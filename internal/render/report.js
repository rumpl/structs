(() => {
  'use strict';

  const search = document.getElementById('search');
  const buttons = [...document.querySelectorAll('[data-filter]')];
  const packages = [...document.querySelectorAll('[data-package]')];
  const sections = [...document.querySelectorAll('[data-type-section]')];
  const tocEntries = new Map([...document.querySelectorAll('[data-toc-entry]')]
    .map(entry => [entry.dataset.tocEntry, entry]));
  const types = [...document.querySelectorAll('article[data-kind]')].map(element => ({
    element,
    kind: element.dataset.kind,
    // Cache text once: typing does not repeatedly read the report's DOM content.
    text: (element.textContent + ' ' + [...element.querySelectorAll('[title]')].map(node => node.title).join(' ') +
      ' ' + element.parentElement.querySelector('h3').textContent +
      ' ' + element.parentElement.querySelector('.directory').textContent).toLocaleLowerCase(),
    toc: tocEntries.get(element.id),
  }));
  // Tree metadata is cached once; updates propagate counts bottom-up instead
  // of rescanning every descendant for every folder in a large report.
  const treeNodes = [...document.querySelectorAll('[data-tree-node]')].map(element => ({
    element,
    count: element.querySelector(':scope > summary [data-tree-count]'),
    children: [...element.querySelector(':scope > .tree-children').children].map(child =>
      child.matches('[data-toc-entry]') ? child : child.querySelector(':scope > [data-tree-node]')),
  }));
  let filter = 'all';
  let beforeSearch = null;

  function revealTreeEntry(entry) {
    if (!entry) return;
    for (let node = entry.parentElement; node; node = node.parentElement) {
      if (node.matches('[data-tree-node]')) node.open = true;
    }
  }

  function markCurrent() {
    const id = window.location.hash.slice(1);
    for (const [target, entry] of tocEntries) {
      const link = entry.querySelector('a');
      if (target === id) link.setAttribute('aria-current', 'location');
      else link.removeAttribute('aria-current');
    }
    revealTreeEntry(tocEntries.get(id));
  }

  function update() {
    const terms = search.value.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
    let visible = 0;
    for (const type of types) {
      const matches = (filter === 'all' || type.kind === filter) &&
        terms.every(term => type.text.includes(term));
      type.element.hidden = !matches;
      type.toc.hidden = !matches;
      if (matches) visible++;
    }
    for (const group of packages) {
      group.hidden = ![...group.querySelectorAll('article')].some(type => !type.hidden);
    }
    for (const section of sections) {
      const hasTypes = section.querySelector('article') !== null;
      const hasMatches = [...section.querySelectorAll('[data-package]')].some(group => !group.hidden);
      section.hidden = (filter !== 'all' && filter !== section.dataset.typeSection) ||
        (hasTypes && !hasMatches) || (!hasTypes && terms.length > 0);
    }
    const searching = terms.length > 0;
    if (searching && beforeSearch === null) {
      beforeSearch = new Map(treeNodes.map(node => [node.element, node.element.open]));
    }
    const counts = new Map();
    for (const node of [...treeNodes].reverse()) {
      const count = node.children.reduce((sum, child) => sum +
        (child.matches('[data-toc-entry]') ? Number(!child.hidden) : counts.get(child)), 0);
      counts.set(node.element, count);
      node.element.hidden = count === 0;
      node.count.textContent = count;
      if (searching && count > 0) node.element.open = true;
      else if (!searching && beforeSearch !== null) node.element.open = beforeSearch.get(node.element);
    }
    if (!searching) beforeSearch = null;
    for (const button of buttons) {
      button.setAttribute('aria-pressed', String(button.dataset.filter === filter));
    }
    document.getElementById('result-count').textContent = visible + ' of ' + types.length + ' types';
    document.getElementById('no-results').hidden = visible > 0 || types.length === 0;
  }

  // A shared link must reveal its target even when the current filters hide it.
  function revealHash() {
    const id = window.location.hash.slice(1);
    const target = document.getElementById(id);
    markCurrent();
    if (!target || (!target.matches('article[data-kind]') && !target.matches('[data-type-section] > h2'))) return;
    if (target.hidden || target.closest('[hidden]')) {
      filter = target.dataset.kind || target.parentElement.dataset.typeSection;
      search.value = '';
      update();
    }
    markCurrent();
    target.scrollIntoView();
    if (target.matches('article')) target.focus({ preventScroll: true });
  }

  search.addEventListener('input', update);
  document.getElementById('clear').addEventListener('click', () => {
    search.value = '';
    update();
    search.focus();
  });
  for (const button of buttons) {
    button.addEventListener('click', () => {
      filter = button.dataset.filter;
      update();
    });
  }
  document.getElementById('tree-expand').addEventListener('click', () => {
    for (const node of treeNodes) if (!node.element.hidden) node.element.open = true;
  });
  document.getElementById('tree-collapse').addEventListener('click', () => {
    for (const node of treeNodes) node.element.open = false;
  });
  document.getElementById('tree-controls').hidden = false;
  window.addEventListener('hashchange', revealHash);
  document.getElementById('controls').hidden = false;
  update();
  revealHash();
})();
