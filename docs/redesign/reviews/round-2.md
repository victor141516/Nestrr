# Revisión UX/QA independiente · ronda 2

## Resumen

La ronda 2 mejora de forma material la cobertura y la calidad del contenedor de revisión: las 32 capturas cargan, hay 155 hotspots con destinos válidos, los seis recorridos guiados pueden alcanzar su último paso, el selector vuelve a ser autoritativo, el historial se deshabilita correctamente, las áreas de interacción superpuestas miden al menos 44×44 px y existe una ficha semántica utilizable en todas las pantallas. La serie ya dispone de player propio en escritorio y móvil, la película diferencia remoto/aceptado/disponible, las rutas físicas están en inglés y hay variantes accionables para Plarr y qBittorrent degradados.

Todavía no debe considerarse aprobada para implementación visual literal. Permanecen contradicciones que afectan a los recorridos centrales: varias láminas siguen mezclando estados incompatibles; el flujo de película cambia de `Bajo la Aurora` a `Órbita Ciega`; la rama episódica salta entre S01E01, S01E02 y S01E05; y los estados remoto/pendiente no tienen variante móvil. Además, no existe una variante degradada específica de Jackett y el hotspot móvil `Enviar resultado` sigue colocado sobre la fila ya enviada.

Resultado: **0 P0, 5 P1, 3 P2 y 1 P3**. **Queda feedback relevante y se justifica una tercera ronda** antes de congelar la UI.

## Evidencia probada

- Leídos íntegramente `product-flows.md`, `visual-spec.md`, `fixture-state.json`, `screen-manifest.json`, `reviews/round-1.md` y los cinco archivos de `prototype/`.
- Prototipo real probado en `https://irsouate.tailfbddd.ts.net:8445/prototype/` con navegador.
- `node docs/redesign/prototype/verify.mjs`: `OK: 32 pantallas, 155 hotspots, 6 recorridos.`
- Recorridas las 32 opciones del selector: todas cargan una imagen con dimensiones naturales, al menos un hotspot y ficha semántica no vacía.
- Sin warnings ni errores de consola después de los recorridos y la recarga posterior a la corrección de suplantación.
- Recorridos de escritorio completados:
  - acceso: `S00 → S01 → S02 → S03 → S04 → S06`;
  - película: `S03 → S04 → S06-remote → S11 → S06-pending → S05 → S06 → S09`;
  - serie: `S03 → S07 → S09-episode → S10 → S07 → S11`;
  - administración: `S03 → S12 → S13 → S03 → S12 → S03`;
  - conexión lenta: `S06 → S09 → S09 → S06`;
  - degradación: `S06 → S06-plarr-down → S06 → S06-torrents-down → S06`.
- La corrección de la banda de suplantación se aplicó durante esta revisión y se revalidó: permanece visible en S03, ya no intercepta Ajustes y desaparece tras volver al administrador.
- Recorrido de serie móvil completado sin saltos de viewport: `S03-mobile → S07-mobile → S09-episode-mobile → S10-mobile → S07-mobile → S11-mobile`.
- Recorrido de película móvil observado: `S03-mobile → S04-mobile → S06-remote (escritorio) → S11-mobile → S06-pending (escritorio) → S05-mobile → S06-mobile → S09-mobile`.
- Breakpoints probados: 320×800, 390×844, 768×1024 y 1440×900. En los cuatro casos `scrollWidth === clientWidth`; hotspot mínimo 44×44 px.
- A 320 px el recorrido guiado comienza contraído y el escenario empieza en `y≈518 px`, dentro del primer viewport.
- Selector probado `S11-mobile → S11`: respeta explícitamente la vista de escritorio.
- Historial probado `S04 → atrás S03 → adelante S04`; Adelante queda deshabilitado al final.
- Panel Información: `role=dialog`, foco inicial en Cerrar, Escape cierra y devuelve el foco a Información.

## Comparación expresa con la ronda 1

| Hallazgo ronda 1 | Estado ronda 2 | Evidencia |
| --- | --- | --- |
| P1.1 Estados no simulables | **Abierto** | S01/S02/S04/S08/S11/S12/S13 continúan mostrando a la vez normal, éxito, error, menú o diálogo; los hotspots de mismo destino sólo producen toast. |
| P1.2 Flujo película contradictorio | **Parcial** | Existen remoto y pendiente, pero S04 abre la tarjeta `Bajo la Aurora` y aterriza en `Órbita Ciega`; S03 empieza con qBittorrent degradado y el flujo pasa a torrent sano sin transición. |
| P1.3 Fixture Puerto Umbral | **Parcial** | Se corrigieron Parcial, S01E03 ausente y S01E05→S01E06; S05 aún marca S01E02 como `Visto` mientras S07 lo muestra con 18 min de progreso, y los agregados de temporada/serie no son coherentes. |
| P1.4 Player rompe contexto serie | **Parcial** | Hay player de episodio, pero `Reproducir S01E01` abre el player de S01E02 y finalizar S01E02 abre autoplay de S01E05. |
| P1.5 Suplantación no persistente | **Cerrado durante la revisión** | Banda global visible y recorrido exacto revalidado sin intercepción. |
| P1.6 Móvil vuelve a escritorio | **Parcial** | La preferencia móvil se conserva, pero no hay `S06-remote-mobile` ni `S06-pending-mobile`; el flujo salta dos veces a escritorio. |
| P1.7 Shell móvil inconsistente | **Visualmente cerrado / interacción parcial** | La shell usa Inicio/Buscar/Descargado y Estado de forma uniforme; el avatar de S03-mobile no tiene hotspot y no permite abrir Ajustes. |
| P1.8 Degradados no accionables | **Parcial** | Plarr y qBittorrent tienen variantes correctas; falta Jackett degradado y el cambio a degradado sólo se alcanza desde el paso del recorrido. |
| P2.1 Selector no autoritativo | **Cerrado** | Seleccionar escritorio desde móvil produce S11/escritorio. |
| P2.2 Hotspots de 40 px | **Cerrado** | Mínimo medido: 44×44 px en cuatro breakpoints. |
| P2.3 Hotspots móviles imprecisos | **Abierto** | `S11-mobile/Enviar resultado` sigue en `y=52.2%`, sobre la primera fila que ya dice `Enviado`; el botón `Enviar` visible está sensiblemente más abajo. |
| P2.4 Sin contenido semántico | **Cerrado** | Todas las pantallas tienen resumen y hechos; las móviles heredan el contrato de su variante. |
| P2.5 Panel de flujo dominante | **Cerrado** | A ≤720 px comienza contraído. |
| P2.6 Flujo activo obsoleto | **Cerrado** | Elegir pantalla libre limpia el flujo y oculta su progreso. |
| P2.7 Sin bootstrap móvil | **Cerrado** | S01-mobile existe y enlaza con S02-mobile/S03-mobile. |
| P3.1 Adelante siempre activo | **Cerrado** | Se activa tras Atrás y se desactiva al volver al extremo. |
| P3.2 Información no es diálogo | **Cerrado** | Rol, foco, Escape, trampa y retorno comprobados. |
| P3.3 S00 sin affordance | **Cerrado** | Hay un selector explícito de condición inicial. |

## Criterios de aceptación de la sección 17

| # | Resultado | Evidencia |
| ---: | --- | --- |
| 1 | Parcial | La ruta técnica de acceso funciona en escritorio y existen bootstrap/login móviles, pero sus estados de envío/error siguen siendo especímenes simultáneos, no variantes activables. |
| 2 | Parcial | Remoto → torrent → pendiente → biblioteca → disponible → play ya existe, pero el contenido cambia de `Bajo la Aurora` a `Órbita Ciega` y la salud de qBittorrent cambia sin explicación. |
| 3 | No conforme | El camino clicable mezcla E01→player E02→autoplay E05→siguiente E06 y después vuelve manualmente a E03 ausente; no representa una sola historia “episodio disponible → siguiente ausente → torrent”. |
| 4 | Parcial | Reanudar → 240p → cerrar se recorre, pero la calidad/progreso no cambian en una variante observable; se anuncia por toast sobre la misma PNG. |
| 5 | Conforme | S05 muestra títulos localizados y rutas `Blind Orbit`, `Threshold Harbor`, `The Last Aurora` y `North Vector` en inglés. |
| 6 | Parcial | El recorrido admin de escritorio es completo y la suplantación ya persiste; los diálogos/errores siguen simultáneos y el Inicio móvil no permite abrir Ajustes desde el avatar. |
| 7 | Parcial | Plarr y qBittorrent respetan la matriz, pero no se representa Jackett degradado aunque exista en `fixture-state.json`. |
| 8 | No conforme | Serie móvil funciona de extremo a extremo, pero película remota/pendiente cae a escritorio y administración móvil no tiene entrada accionable desde el avatar de Inicio. |

## Hallazgos P0

Ninguno.

## Hallazgos P1

### P1.1 · Los estados incompatibles siguen fijados en una misma captura

La recomendación principal de la ronda 1 no se ha implementado como modelo de estado. Ejemplos verificables:

- S04 muestra resultados, `Comprobando 3 de 8` y `Sin coincidencias` al mismo tiempo; además incluye `Puerto Umbral` con Películas seleccionado.
- S11 contiene simultáneamente una fila enviada, una lista y un fallo reintentable.
- S12 combina banda de suplantación, menú abierto, preferencias guardadas y confirmación destructiva abierta.
- S13 combina formulario vacío, error de duplicado, menú de acciones y diálogo de reset.
- S06-remote, S06-pending y S06-plarr-down mantienen visible el toast `Metadatos actualizados` aunque se hayan abierto como estado inicial.

Los hotspots de mismo destino sólo superponen un mensaje. No permiten validar bloqueo, foco, busy, error, restauración ni jerarquía entre estados. Esta limitación afecta directamente a autenticación, búsqueda, torrent, ajustes y administración.

### P1.2 · El flujo de película conserva estados, pero no la identidad ni la salud del contexto

La única tarjeta remota accionable de S04 es `Bajo la Aurora`; su hotspot abre S06-remote, cuya película es `Órbita Ciega`. El recorrido registrado es técnicamente correcto por IDs, pero el usuario cambia de contenido al pulsar la tarjeta. Además, S03 presenta qBittorrent degradado y el flujo llega a S06-remote/S11 con qBittorrent sano sin una recuperación observable.

El flujo debe usar el mismo título de extremo a extremo y fijar o transicionar explícitamente la salud de servicios.

### P1.3 · El fixture episódico y el recorrido guiado aún mezclan ramas

Persisten cuatro contradicciones:

1. `fixture-state.json` define S01E02 como `available-progress`; S07 muestra 18 min de progreso, pero S05 lo marca `Visto`.
2. El hero de S07 declara 5/8 global y Temporada 1 declara 5/8, mientras Temporada 2 añade 0/6 en la misma lámina; la agregación global deja de ser interpretable.
3. El hotspot `Reproducir S01E01` abre S09-episode, rotulado `S01E02 · Marea muerta`.
4. Finalizar ese player S01E02 abre S10 con episodio actual S01E05 y siguiente S01E06.

S10 es una rama válida de “siguiente disponible”, pero no puede encadenarse directamente al player S01E02 ni sustituye la rama de aceptación “siguiente ausente → S07 focalizado → torrent”.

### P1.4 · Los recorridos críticos no son todavía íntegramente móviles

La preferencia móvil ya no se pierde, pero `S06-remote` y `S06-pending` no tienen composición móvil. El camino real alterna Móvil → Escritorio → Móvil → Escritorio → Móvil. En administración, S03-mobile dibuja avatar, pero sus cinco hotspots no incluyen Ajustes/usuario; por tanto el flujo sólo puede continuar saltando con los controles del revisor, no usando la UI mostrada.

### P1.5 · La matriz degradada no cubre Jackett

`fixture-state.json` diferencia `jackett-down` y `qbittorrent-down`, y el criterio 7 exige cada combinación. Sólo existe `S06-torrents-down`, etiquetada específicamente como qBittorrent degradado. Aunque ambas caídas oculten Torrent, falta demostrar el estado/mensaje seguro de Jackett y su recuperación localizada.

## Hallazgos P2

### P2.1 · `Enviar resultado` en S11-mobile sigue sobre la fila equivocada

La región está en `y=52.2%` y activa la transición a pendiente sobre la primera fila, cuyo estado visible es `Enviado`. El botón `Enviar` de la segunda fila está aproximadamente en el último tercio previo al teclado. Semántica y coordenada vuelven a discrepar.

### P2.2 · Los cambios a estados degradados dependen del panel del revisor

Desde S06 no hay una acción o control de escenario que lleve a `S06-plarr-down` o `S06-torrents-down`; para probarlos hay que pulsar los pasos 2 y 4 del recorrido guiado. Es válido que una caída no sea una acción del producto, pero el simulador debería exponer una condición de servicio explícita, como S00 hace con la sesión, para que el recorrido no parezca una navegación de usuario.

### P2.3 · Los anuncios dinámicos no distinguen éxito de error

`showActionState()` duplica cualquier mutación en un toast `role=status`. No existe una variante `role=alert` para los errores simulados ni estados busy/disabled asociados. La ficha semántica resuelve el contenido estático, pero no la semántica de mutaciones que P1.1 aún representa mediante mensajes.

## Hallazgos P3

### P3.1 · Parte de la barra del prototipo conserva objetivos de 40 px

Los hotspots de producto ya cumplen 44 px, pero historial, acciones de la toolbar, selects y cierre de Información conservan `2.5rem`/40 px. Es deuda del contenedor de revisión, no de las pantallas de Nestrr.

## Cambios concretos recomendados para la ronda 3

1. Separar como mínimo variantes activables de S02, S04, S11, S12 y S13 (ready/loading/success/error/dialog), y usar esas variantes en los recorridos en vez de toast sobre una lámina compuesta.
2. Sustituir en S04 la tarjeta remota por `Órbita Ciega` o cambiar las cuatro variantes S06 y el fixture a `Bajo la Aurora`; mantener una sola identidad.
3. Fijar S01E02 como progreso parcial en S05/S07/player y corregir el agregado de Puerto Umbral para que temporadas y total de serie sumen lo mismo.
4. Separar dos ramas episódicas: player S01E05 → S10 → S01E06 disponible; y player de un episodio cuyo siguiente sea S01E03 ausente → S07 con E03 enfocado → S11. No enlazar E02 directamente con S10/E05.
5. Parametrizar o duplicar el player para que `Reproducir S01E01` no abra S01E02.
6. Crear `S06-remote-mobile` y `S06-pending-mobile`, y añadir hotspot de avatar/Ajustes en S03-mobile.
7. Añadir `S06-jackett-down` o convertir la variante de torrents en un selector explícito Jackett/qBittorrent con mensajes propios.
8. Mover el hotspot `Enviar resultado` de S11-mobile al botón de la segunda fila.
9. Añadir controles de escenario para salud de servicios y tipos semánticos distintos para status/alert/busy.

## Veredicto final

**No aprobado todavía para congelar e implementar tal cual.** La ronda 2 cierra la mayoría de defectos del contenedor y aporta la arquitectura visual necesaria, pero aún hay discontinuidades de identidad, episodio, estado y viewport en los recorridos que el usuario utilizará para validar la implementación. **Sí queda feedback relevante que justifica una tercera ronda**, centrada en estados activables, coherencia de fixture/ramas y continuidad móvil. Después de esos cambios debería bastar una revisión breve de regresión, no otra exploración completa.
