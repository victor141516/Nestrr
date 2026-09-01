(async function () {
  "use strict";

  const data = window.NESTRR_PROTOTYPE;
  const elements = {
    image: document.querySelector("#screen-image"),
    frame: document.querySelector("#screen-frame"),
    layer: document.querySelector("#hotspot-layer"),
    select: document.querySelector("#screen-select"),
    flowSelect: document.querySelector("#flow-select"),
    flowProgress: document.querySelector("#flow-progress"),
    flowTitle: document.querySelector("#flow-title"),
    flowDescription: document.querySelector("#flow-description"),
    flowSteps: document.querySelector("#flow-steps"),
    scenarioControls: document.querySelector("#scenario-controls"),
    title: document.querySelector("#screen-title"),
    kicker: document.querySelector("#screen-kicker"),
    description: document.querySelector("#screen-description"),
    hotspotCount: document.querySelector("#hotspot-count"),
    viewportLabel: document.querySelector("#viewport-label"),
    back: document.querySelector("#back-button"),
    forward: document.querySelector("#forward-button"),
    variant: document.querySelector("#variant-button"),
    hotspots: document.querySelector("#hotspots-button"),
    info: document.querySelector("#info-button"),
    infoPanel: document.querySelector("#info-panel"),
    closeInfo: document.querySelector("#close-info"),
    actionState: document.querySelector("#action-state"),
    impersonation: document.querySelector("#impersonation-overlay"),
    stopImpersonation: document.querySelector("#stop-impersonation"),
    semanticSummary: document.querySelector("#semantic-summary"),
    semanticFacts: document.querySelector("#semantic-facts"),
    announcement: document.querySelector("#screen-announcement"),
    toast: document.querySelector("#toast")
  };

  let manifest;
  try {
    const response = await fetch("../screen-manifest.json", { cache: "no-store" });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    manifest = await response.json();
  } catch (error) {
    document.body.innerHTML = `<main class="prototype-layout"><h1>No se pudo abrir el prototipo</h1><p>Sirve <code>docs/redesign</code> mediante HTTP y abre <code>/prototype/</code>.</p><pre>${String(error)}</pre></main>`;
    return;
  }

  const screens = new Map(manifest.screens.map((screen) => [screen.id, screen]));
  const isMobileScreen = (screen) => Boolean(screen?.id.endsWith("-mobile"));
  const desktopIdFor = (screen) => isMobileScreen(screen) ? screen.id.slice(0, -"-mobile".length) : screen.id;
  const mobileByDesktop = new Map(manifest.screens.filter(isMobileScreen).map((screen) => [desktopIdFor(screen), screen.id]));
  const state = {
    current: null,
    showHotspots: true,
    mobilePreferred: false,
    activeFlow: "",
    flowIndex: 0,
    historyIndex: 0,
    maxHistoryIndex: 0,
    flags: { impersonating: false },
    toastTimer: null,
    actionTimer: null,
    infoReturnFocus: null
  };

  function screenLabel(screen) {
    return `${screen.id} · ${screen.title}`;
  }

  function populateScreenSelect() {
    const desktop = manifest.screens.filter((screen) => !isMobileScreen(screen));
    const mobile = manifest.screens.filter(isMobileScreen);
    const desktopGroup = document.createElement("optgroup");
    desktopGroup.label = "Escritorio";
    desktop.forEach((screen) => desktopGroup.append(new Option(screenLabel(screen), screen.id)));
    const mobileGroup = document.createElement("optgroup");
    mobileGroup.label = "Móvil";
    mobile.forEach((screen) => mobileGroup.append(new Option(screenLabel(screen), screen.id)));
    elements.select.replaceChildren(desktopGroup, mobileGroup);
  }

  function preferredTarget(target) {
    if (!state.mobilePreferred || target.endsWith("-mobile")) return target;
    return mobileByDesktop.get(target) || target;
  }

  function updateFlowForNavigation(target) {
    if (!state.activeFlow) return;
    const flow = data.flows[state.activeFlow];
    const targetScreen = screens.get(target);
    const desktopTarget = desktopIdFor(targetScreen);
    for (let index = state.flowIndex + 1; index < flow.steps.length; index += 1) {
      if (flow.steps[index] === desktopTarget) {
        state.flowIndex = index;
        return;
      }
    }
    const existing = flow.steps.indexOf(desktopTarget);
    if (existing !== -1) {
      state.flowIndex = existing;
      return;
    }
    state.activeFlow = "";
    state.flowIndex = 0;
    elements.flowSelect.value = "";
  }

  function navigate(target, options = {}) {
    const resolved = options.exact ? target : preferredTarget(target);
    if (!screens.has(resolved)) {
      showToast(`Destino no disponible: ${resolved}`);
      return;
    }
    updateFlowForNavigation(resolved);
    const url = new URL(location.href);
    url.hash = resolved;
    if (options.replace) {
      state.historyIndex = Number(history.state?.index || 0);
      state.maxHistoryIndex = Math.max(state.maxHistoryIndex, state.historyIndex);
      history.replaceState({ screen: resolved, index: state.historyIndex }, "", url);
    } else {
      state.historyIndex += 1;
      state.maxHistoryIndex = state.historyIndex;
      history.pushState({ screen: resolved, index: state.historyIndex }, "", url);
    }
    render(resolved);
  }

  function showToast(message, tone = "status") {
    clearTimeout(state.toastTimer);
    elements.toast.textContent = message;
    elements.toast.setAttribute("role", tone === "alert" ? "alert" : "status");
    elements.toast.classList.toggle("error", tone === "alert");
    elements.toast.hidden = false;
    state.toastTimer = setTimeout(() => { elements.toast.hidden = true; }, 3200);
  }

  function showActionState(message, tone = "status") {
    clearTimeout(state.actionTimer);
    elements.actionState.textContent = message;
    elements.actionState.setAttribute("role", tone === "alert" ? "alert" : "status");
    elements.actionState.classList.toggle("error", tone === "alert");
    elements.actionState.hidden = false;
    state.actionTimer = setTimeout(() => { elements.actionState.hidden = true; }, 4200);
    showToast(message, tone);
  }

  function makeHotspot(hotspot) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "hotspot";
    button.setAttribute("aria-label", hotspot.action);
    button.style.left = `${hotspot.x}%`;
    button.style.top = `${hotspot.y}%`;
    button.style.width = `${hotspot.w}%`;
    button.style.height = `${hotspot.h}%`;
    button.addEventListener("click", () => {
      if (hotspot.setFlag) state.flags[hotspot.setFlag] = true;
      if (hotspot.clearFlag) state.flags[hotspot.clearFlag] = false;
      const target = preferredTarget(hotspot.target);
      if (hotspot.message || target === state.current) {
        showActionState(hotspot.message || `${hotspot.action}: estado simulado en esta pantalla.`, hotspot.tone);
      }
      if (target !== state.current) navigate(target);
      else {
        updateFlowForNavigation(target);
        renderFlow();
      }
    });
    return button;
  }

  function renderFlow() {
    if (!state.activeFlow) {
      elements.flowProgress.hidden = true;
      return;
    }
    const flow = data.flows[state.activeFlow];
    elements.flowProgress.hidden = false;
    elements.flowTitle.textContent = flow.title;
    elements.flowDescription.textContent = flow.description;
    elements.flowSteps.replaceChildren(...flow.steps.map((screenId, index) => {
      const item = document.createElement("li");
      if (index < state.flowIndex) item.className = "complete";
      if (index === state.flowIndex) item.className = "current";
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = `${index + 1}. ${screens.get(screenId)?.title || screenId}`;
      button.setAttribute("aria-current", index === state.flowIndex ? "step" : "false");
      button.addEventListener("click", () => {
        state.flowIndex = index;
        navigate(screenId);
      });
      item.append(button);
      return item;
    }));
  }

  function renderSemantic(screen) {
    const semantic = data.semantic[screen.id] || data.semantic[screen.variantOf] || {
      summary: screen.description,
      facts: { Pantalla: screen.title }
    };
    elements.semanticSummary.textContent = semantic.summary;
    const entries = Object.entries(semantic.facts || {});
    elements.semanticFacts.replaceChildren(...entries.flatMap(([term, description]) => {
      const dt = document.createElement("dt");
      const dd = document.createElement("dd");
      dt.textContent = term;
      dd.textContent = description;
      return [dt, dd];
    }));
  }

  function renderScenarioControls(screen) {
    const scenario = data.scenarios?.[screen.id] || data.scenarios?.[screen.variantOf];
    elements.scenarioControls.hidden = !scenario;
    if (!scenario) {
      elements.scenarioControls.replaceChildren();
      return;
    }
    const label = document.createElement("strong");
    label.textContent = scenario.label;
    const options = document.createElement("div");
    options.replaceChildren(...scenario.options.filter((option) => screens.has(option.target)).map((option) => {
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.target = option.target;
      button.textContent = option.label;
      if (option.target === screen.id) button.setAttribute("aria-current", "true");
      return button;
    }));
    elements.scenarioControls.replaceChildren(label, options);
  }

  function render(id) {
    const screen = screens.get(id) || screens.get("S00");
    state.current = screen.id;
    const isMobile = isMobileScreen(screen);
    if (isMobile) state.mobilePreferred = true;
    const hotspots = data.hotspots[screen.id] || [];
    elements.select.value = screen.id;
    elements.title.textContent = screen.title;
    elements.kicker.textContent = `Pantalla ${screen.id}`;
    elements.description.textContent = screen.description;
    elements.image.src = `../${screen.file}`;
    elements.image.alt = `Captura conceptual: ${screen.title}. ${screen.description}`;
    elements.frame.classList.toggle("mobile", isMobile);
    const busyState = /-(loading|sending)(-mobile)?$/.test(screen.id);
    const errorState = /-error(-mobile)?$/.test(screen.id);
    const successState = /-(accepted|saved)(-mobile)?$/.test(screen.id);
    elements.frame.setAttribute("aria-busy", String(busyState));
    elements.announcement.setAttribute("role", errorState ? "alert" : "status");
    elements.announcement.textContent = busyState
      ? `${screen.title}: operación en curso.`
      : errorState
        ? `${screen.title}: se produjo un error recuperable.`
        : successState
          ? `${screen.title}: operación completada.`
          : "";
    elements.frame.classList.toggle("show-hotspots", state.showHotspots);
    elements.layer.replaceChildren(...hotspots.map(makeHotspot));
    elements.hotspotCount.textContent = `${hotspots.length} ${hotspots.length === 1 ? "zona" : "zonas"}`;
    elements.viewportLabel.textContent = isMobile ? "Móvil" : "Escritorio";
    elements.variant.textContent = isMobile ? "Vista escritorio" : "Vista móvil";
    elements.variant.setAttribute("aria-pressed", String(isMobile));
    elements.actionState.hidden = true;
    renderScenarioControls(screen);
    elements.impersonation.hidden = !state.flags.impersonating;
    elements.back.disabled = state.historyIndex <= 0;
    elements.forward.disabled = state.historyIndex >= state.maxHistoryIndex;
    renderSemantic(screen);
    renderFlow();
    document.title = `${screen.id} · ${screen.title} · Nestrr`;
  }

  function toggleVariant() {
    const screen = screens.get(state.current);
    if (isMobileScreen(screen)) {
      state.mobilePreferred = false;
      navigate(desktopIdFor(screen));
      return;
    }
    const mobile = mobileByDesktop.get(screen.id);
    if (!mobile) {
      showToast("Esta pantalla no tiene captura móvil propia; conserva la versión de escritorio.");
      return;
    }
    state.mobilePreferred = true;
    navigate(mobile);
  }

  populateScreenSelect();
  elements.select.addEventListener("change", () => {
    const selected = elements.select.value;
    state.activeFlow = "";
    state.flowIndex = 0;
    elements.flowSelect.value = "";
    state.mobilePreferred = selected.endsWith("-mobile");
    navigate(selected, { exact: true });
  });
  elements.flowSelect.addEventListener("change", () => {
    state.activeFlow = elements.flowSelect.value;
    state.flowIndex = 0;
    if (!state.activeFlow) {
      renderFlow();
      return;
    }
    elements.flowProgress.open = !matchMedia("(max-width: 720px)").matches;
    navigate(data.flows[state.activeFlow].steps[0]);
  });
  elements.back.addEventListener("click", () => history.back());
  elements.forward.addEventListener("click", () => history.forward());
  elements.variant.addEventListener("click", toggleVariant);
  elements.hotspots.addEventListener("click", () => {
    state.showHotspots = !state.showHotspots;
    elements.frame.classList.toggle("show-hotspots", state.showHotspots);
    elements.hotspots.classList.toggle("active", state.showHotspots);
    elements.hotspots.setAttribute("aria-pressed", String(state.showHotspots));
    elements.hotspots.textContent = state.showHotspots ? "Zonas visibles" : "Mostrar zonas";
  });
  elements.info.addEventListener("click", () => {
    const open = elements.infoPanel.hidden;
    elements.infoPanel.hidden = !open;
    elements.info.setAttribute("aria-expanded", String(open));
    if (open) {
      state.infoReturnFocus = document.activeElement;
      elements.closeInfo.focus();
    }
  });
  elements.closeInfo.addEventListener("click", () => {
    elements.infoPanel.hidden = true;
    elements.info.setAttribute("aria-expanded", "false");
    (state.infoReturnFocus || elements.info).focus();
  });
  elements.scenarioControls.addEventListener("click", (event) => {
    const button = event.target.closest("button[data-target]");
    if (button) navigate(button.dataset.target, { exact: true });
  });
  elements.stopImpersonation.addEventListener("click", () => {
    state.flags.impersonating = false;
    showToast("Has vuelto a la sesión de administrador.");
    navigate("S03");
  });
  window.addEventListener("popstate", (event) => {
    state.historyIndex = Number(event.state?.index || 0);
    const id = location.hash.slice(1);
    render(screens.has(id) ? id : "S00");
  });
  window.addEventListener("keydown", (event) => {
    if (event.target.matches("input, select, textarea")) return;
    if (event.key.toLowerCase() === "h") elements.hotspots.click();
    if (event.key === "ArrowLeft" && event.altKey) history.back();
    if (event.key === "ArrowRight" && event.altKey) history.forward();
    if (event.key === "Escape" && !elements.infoPanel.hidden) elements.closeInfo.click();
    if (event.key === "Tab" && !elements.infoPanel.hidden) {
      const focusable = [...elements.infoPanel.querySelectorAll("button, [href], select, input, [tabindex]:not([tabindex='-1'])")].filter((item) => !item.disabled);
      if (!focusable.length) return;
      const first = focusable[0], last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    }
  });

  const initial = location.hash.slice(1);
  navigate(screens.has(initial) ? initial : "S00", { replace: true });
})();
