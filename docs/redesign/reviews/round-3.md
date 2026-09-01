# Revisión UX/QA independiente · ronda 3

## Resumen

La ronda 3 cierra de forma convincente los problemas visuales centrales de la ronda 2. Las 85 pantallas existen como estados individuales, las siete familias de recorrido son navegables, la identidad de Órbita Ciega ya no cambia, Puerto Umbral conserva un fixture coherente y los caminos de película y serie funcionan de extremo a extremo tanto en escritorio como en móvil. También quedan resueltos el botón Enviar móvil y la simulación explícita de Plarr, Jackett y qBittorrent.

No obstante, apareció una regresión funcional de suplantación que impide aprobar todavía el prototipo: la banda global desaparece al entrar en Ajustes y, tras la separación de estados de S12, tampoco queda allí una acción para volver al administrador. Además quedan dos deudas semánticas P2 del contenedor, localizadas y corregibles sin rehacer imágenes.

Resultado: **0 P0, 1 P1, 2 P2 y 1 P3**. **Sí queda feedback relevante que justifica una ronda 4 breve de regresión**, centrada en la suplantación y en la semántica de degradación/estados. No hace falta otra exploración visual completa ni regenerar el sistema de pantallas.

## Evidencia probada

- Leídos íntegramente `product-flows.md`, `fixture-state.json`, `visual-spec.md`, `screen-manifest.json`, `reviews/round-1.md`, `reviews/round-2.md` y los cinco archivos de `prototype/`.
- Prototipo real probado en `https://irsouate.tailfbddd.ts.net:8445/prototype/?review=round3` con los scripts `v=6`.
- `node docs/redesign/prototype/verify.mjs`: `OK: 85 pantallas, 193 hotspots, 7 recorridos.`
- Recorridas las 85 opciones del selector: 85/85 cargaron imagen con dimensiones naturales, al menos un hotspot y ficha semántica no vacía.
- Sin warnings ni errores de consola.
- Breakpoints probados: 320×800, 390×844, 768×1024 y 1440×900. En todos `scrollWidth === clientWidth` y el objetivo interactivo visible mínimo mide 44×44 px.
- Película escritorio completada por UI: `S03 → S04 → S06-remote → S11-ready → S11-loading → S11 → S11-sending → S11-accepted → S06-pending → S05 → S06 → S09`.
- Película móvil completada sin saltos de viewport: `S03-mobile → S04-mobile → S06-remote-mobile → S11-ready-mobile → S11-loading-mobile → S11-mobile → S11-sending-mobile → S11-accepted-mobile → S06-pending-mobile → S05-mobile → S06-mobile → S09-mobile`.
- Rama episódica escritorio y móvil: `S07 → E01 → E02 → E03 no disponible → S07 → torrent`.
- Autoplay independiente escritorio y móvil: `E05 → S10 → E06 · La baliza`.
- Conexión lenta: `S06 → S09 → reintento en S09 → S06`; el mensaje simulado se expone como `role=status`.
- Servicios degradados probados con el control de escenario: Plarr deja únicamente Torrent; Jackett y qBittorrent conservan Play y eliminan Torrent; todos recuperan de forma localizada.
- En S11 móvil, el hotspot Enviar está en `y=52,2 %`, mide al menos 44 px, se superpone al primer botón Enviar visible y lleva a `S11-sending-mobile`, después a `S11-accepted-mobile`.
- Anuncios same-screen comprobados: enumeración de biblioteca usa `role=status`; error de Temporada 2 usa `role=alert`.

## Comparación uno por uno con los 5 P1 de la ronda 2

| Hallazgo ronda 2 | Estado ronda 3 | Evidencia |
| --- | --- | --- |
| P1.1 Estados incompatibles en una misma captura | **Cerrado visualmente** | S01/S02/S04/S08/S11/S12/S13 tienen variantes ready/loading/empty/error/sending/accepted/menu/dialog separadas en escritorio y móvil. No quedan composites legacy. |
| P1.2 Identidad y salud incoherentes en película | **Cerrado** | S04 muestra Órbita Ciega como remota y Plarr/Jackett/qBittorrent sanos; la misma identidad continúa por torrent, pendiente, biblioteca, disponible y player. |
| P1.3 Fixture episódico mezclado | **Cerrado** | S05 muestra S01E02 al 38 % y 18:32, no visto; S07 mantiene 5/8 disponibles y 3/8 vistos, E03 ausente y Temporada 2 en error sin denominador inventado. |
| P1.4 Recorridos críticos no íntegramente móviles | **Cerrado** | Remoto y pendiente tienen variantes móviles; la cadena completa de película permanece móvil. S03-mobile tiene hotspot de avatar y permite llegar a Ajustes/Administración. |
| P1.5 Falta Jackett degradado | **Cerrado visual e interactivamente** | `S06-jackett-down` y su variante móvil muestran Jackett degradado, mantienen Play/descarga y retiran Torrent; el escenario expone la condición y su recuperación. |

## Comparación uno por uno con los 3 P2 de la ronda 2

| Hallazgo ronda 2 | Estado ronda 3 | Evidencia |
| --- | --- | --- |
| P2.1 Enviar móvil sobre la fila equivocada | **Cerrado** | La captura ya no contiene una fila preenviada; el hotspot está sobre el botón Enviar real de la primera fila y entra en sending → accepted. |
| P2.2 Degradados sólo accesibles desde el recorrido | **Cerrado** | S06 expone `Simular salud de servicios` con Todo sano, Plarr, Jackett y qBittorrent. |
| P2.3 Anuncios dinámicos sin distinguir éxito/error | **Parcial** | Los mensajes same-screen ya usan `status` y `alert`, pero loading/accepted/error que cambian de pantalla no exponen `aria-busy` ni una región status/alert específica; sólo cambia el título dentro de una región polite. |

## Recorridos guiados

| Flujo | Resultado | Evidencia |
| --- | --- | --- |
| Acceso → descubrimiento | **Conforme con una desviación menor** | S00 → bootstrap ready/loading → S03 funciona. Desde S03 el hotspot Buscar salta directamente a resultados; ready/loading siguen disponibles como pasos guiados y estados separados. |
| Película → torrent → biblioteca → play | **Conforme escritorio/móvil** | Identidad, salud y viewport permanecen estables; aceptado no se confunde con disponible. |
| Serie parcial → episodio ausente → torrent | **Conforme escritorio/móvil** | E01 abre player E01, siguiente abre E02, y desde E02 el siguiente E03 ausente vuelve a S07 con torrent. |
| E05 → autoplay → E06 | **Conforme escritorio/móvil** | E05 finaliza en S10; Reproducir ahora abre E06 La baliza, cuyo siguiente está deshabilitado. |
| Administración → suplantar → volver | **No conforme** | La banda aparece en S03 después de Ver como Bruno, pero desaparece en S12 y allí no queda acción Volver al administrador. |
| Conexión lenta | **Conforme como demostrador** | Reintento conserva el mismo player/progreso y anuncia Auto 240p; sigue siendo una simulación raster, apropiada para esta fase. |
| Servicios degradados | **Conforme visualmente; semántica parcial** | La matriz de acciones es correcta y existe control de escenario, pero las fichas de Jackett/qBittorrent heredan la semántica sana de S06. |

## Hallazgos P0

Ninguno.

## Hallazgos P1

### P1.1 · La banda de suplantación desaparece en Ajustes y se pierde la salida

Secuencia reproducida: `S13-member-menu → Ver como Bruno → S03 → Abrir ajustes → S12`.

- En S03, `#impersonation-overlay` es visible.
- En S12 pasa a oculto porque `app.js` excluye explícitamente `S12` y `S12-mobile` al renderizar la banda.
- Las nuevas capturas `s12-settings-ready*.png` ya no incluyen una banda raster propia.
- El override de hotspots de ronda 3 para S12 reemplaza la versión anterior que contenía `Volver al administrador`; la pantalla sólo ofrece Guardar, Eliminar perfil y Abrir menú.

El requisito global de la sección 2.5/4.2 queda roto y el recorrido guiado de administración sólo puede terminar usando los controles externos del revisor o navegando a otra pantalla. Es una regresión funcional relevante, no microtexto.

## Hallazgos P2

### P2.1 · Las fichas semánticas de Jackett y qBittorrent degradados describen el estado sano

El escenario y las imágenes son correctos, pero `renderSemantic()` cae a la ficha base S06 para `S06-jackett-down` y `S06-qbittorrent-down`. La tabla semántica conserva una clave obsoleta `S06-torrents-down`, que ya no coincide con el ID del manifiesto, y no existe entrada específica para Jackett. En la prueba, ambas variantes anunciaron “Detalle de película con disponibilidad compartida y progreso personal separados” en vez de identificar el servicio degradado y la acción retirada.

Esto no bloquea la revisión visual, pero sí impide validar sin imagen el criterio “servicio concreto → dependencia concreta”.

### P2.2 · Loading/accepted/error por navegación aún no tienen semántica de mutación completa

Los mensajes simulados en la misma pantalla sí distinguen `role=status` y `role=alert`, cerrando la mitad principal del hallazgo anterior. Sin embargo:

- S01/S02/S04/S11 loading no marcan el escenario o la región como `aria-busy`;
- entrar en S11 accepted no produce una región `status` específica;
- entrar en S04/S11 error no produce una región `alert` específica.

El título cambia dentro de una región `aria-live=polite`, por lo que existe una señal básica, pero los estados de error no tienen la urgencia ni el contrato semántico indicado en `product-flows.md`.

## Hallazgo P3

### P3.1 · El modo móvil puede filtrarse a un recorrido nuevo

Después de revisar una pantalla móvil, iniciar un recorrido de escritorio desde S00 puede comenzar en S00/S01 de escritorio y resolver el siguiente target hacia `-mobile` porque `mobilePreferred` conserva el valor anterior. Seleccionar explícitamente una pantalla de escritorio lo corrige. Es una deuda del contenedor de revisión, no de la UI de Nestrr, y no justificaría por sí sola otra ronda.

## Aspectos aprobados para implementación

- Dirección visual, jerarquía, shell desktop/móvil, contraste y lenguaje de estados.
- Modelo de estado separado de ready/loading/empty/error/sending/accepted/menu/dialog.
- Fixture de Órbita Ciega y Puerto Umbral.
- Flujos de película, serie, autoplay y servicios degradados.
- Matriz Plarr/Jackett/qBittorrent.
- Rutas físicas canónicas inglesas con títulos de UI localizados.
- Objetivos táctiles, ausencia de overflow y colocación de Enviar móvil.

Las variaciones de microtexto raster indicadas en `visual-spec.md` son P3/P4 y deben convertirse en texto real en frontend; no requieren otra ronda visual.

## Veredicto final

**Todavía no aprobado como prototipo final sin observaciones.** Queda **un P1 relevante**: la suplantación deja de ser global y pierde su salida en Ajustes. Los dos P2 son ajustes del contenedor semántico, no del sistema visual.

La siguiente iteración puede ser muy corta y no necesita regenerar imágenes: mantener la banda overlay en S12/S12-mobile, conservar una acción Volver al administrador, añadir las fichas semánticas degradadas correctas y marcar/anunciar loading, accepted y error. Tras esa corrección, una regresión focalizada debería bastar; no se justifica otra exploración visual completa.
