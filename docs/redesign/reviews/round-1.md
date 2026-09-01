# Revisión UX/QA independiente · ronda 1

## Resumen

El prototipo es técnicamente navegable y el sistema visual es coherente, legible y notablemente más claro que una simple colección de wireframes. Los cinco recorridos guiados llegan a su último paso, las 20 capturas existen, los 96 hotspots tienen nombre accesible y destino válido, y no aparecieron errores de consola. Sin embargo, aún no debe considerarse aprobado para implementación: varias capturas mezclan estados mutuamente excluyentes y los fixtures contradicen la historia que intentan contar, especialmente en descarga, serie/autoplay, suplantación y continuidad móvil.

Resultado: **0 P0, 8 P1, 7 P2 y 3 P3**. Queda feedback relevante y se justifica claramente otra ronda.

## Evidencia probada

- Leídos íntegramente `product-flows.md` (incluidos los ocho criterios de aceptación), `visual-spec.md`, `screen-manifest.json` y todos los archivos de `prototype/`.
- Prueba real en `http://127.0.0.1:4174/prototype/` con navegador. Sin warnings ni errores de consola.
- Recorridos completados mediante hotspots:
  - acceso: `S00 → S01 → S02 → S03 → S04 → S06`;
  - película: `S03 → S04 → S06 → S11 → S05 → S06 → S09`;
  - serie: `S03 → S07 → S09 → S10 → S07 → S11`;
  - administración: `S03 → S12 → S13 → S03 → S12 → S03`;
  - conexión lenta: `S06 → S09 → S09 → S06`.
- Probados los botones Atrás/Adelante: `S04 → S03 → S04`; la navegación funciona.
- Probado el selector en las 20 pantallas. Las 14 capturas de escritorio y las 6 móviles cargan y exponen hotspots con nombres.
- Probado el conmutador escritorio/móvil en S02, S03, S06, S07, S09 y S11; conmuta correctamente cuando se parte de modo escritorio.
- Probados 320×800, 390×844, 768×1024 y 1440×900. No hay overflow horizontal de documento.
- `node docs/redesign/prototype/verify.mjs`: `OK: 20 pantallas, 96 hotspots, 5 recorridos.`
- No aparecen funciones prohibidas: no hay cola/porcentaje de torrents, control de Chiprr, Mi lista, casting, registro público, calidad persistente ni secretos editables.

## Criterios de aceptación de la sección 17

| # | Resultado | Evidencia |
| ---: | --- | --- |
| 1 | Parcial | Bootstrap/login → Inicio → búsqueda → detalle se recorre, pero S01/S02 muestran simultáneamente normal, validación, envío y error en una captura fija. |
| 2 | No conforme | La película ya está disponible y ofrece `Reanudar` antes de abrir torrent; S11 se superpone al mismo detalle disponible y S05 ya contiene el archivo. No se simula realmente remoto → enviado → pendiente → organizado → disponible. |
| 3 | No conforme | S07 marca S01E03 como no descargado, mientras S10 anuncia ese mismo S01E03 como el siguiente disponible y S05 lo muestra ya en disco. Además, el paso S09 intermedio muestra la película Órbita Ciega, no el episodio. |
| 4 | Parcial | Reanudar → 240p → cerrar se recorre y los mensajes dicen conservar progreso, pero no hay cambio de estado verificable; se vuelve a la misma captura fija. |
| 5 | Parcial | S12 explica bien idioma/audio y nombres físicos, pero S05 muestra rutas localizadas (`Movies/Órbita Ciega`, `La Última Aurora`) en vez de demostrar explícitamente nombres canónicos ingleses. |
| 6 | Parcial | Crear/resetear/suplantar/volver tiene camino, pero crear/editar/eliminar son especímenes simultáneos y, tras suplantar, S03 no conserva la banda global; sólo reaparece en S12. |
| 7 | No conforme | Hay ejemplos degradados de qBittorrent, pero no caminos verificables para cada combinación de Plarr y Jackett/qBittorrent ni se demuestra que las acciones se oculten/inhabiliten de acuerdo con la matriz. |
| 8 | No conforme | Sólo hay seis variantes móviles. Desde S03-mobile, Buscar lleva a S04 de escritorio y después a S06 de escritorio. Administración, ajustes, biblioteca, búsqueda, autoplay y bootstrap no pueden recorrerse íntegramente en móvil. |

## Hallazgos P0

Ninguno. No hay pérdida de datos, bloqueo total del prototipo ni destino inválido detectado.

## Hallazgos P1

### P1.1 · Los estados no son realmente simulables

S01, S02, S04, S08, S11, S12 y S13 agrupan a la vez estados incompatibles: listo/enviando/error/éxito, resultados/sin coincidencias, colección con partes/vacía/error, o usuario creado/conflicto. `visual-spec.md` dice que esos especímenes deben activarse como estados separados; el prototipo sólo muestra una PNG fija y los hotspots producen un toast. Esto impide evaluar jerarquía, foco, bloqueo de botones y recuperación real.

### P1.2 · El flujo de película contradice su premisa

S06 muestra Órbita Ciega `Disponible en disco`, 42 % de progreso y `Reanudar` antes del torrent. S11 busca esa misma película desde el detalle disponible y S05 ya la enumera. El recorrido se puede clicar, pero no representa la transición crítica “no descargada → enviada → aún pendiente → disponible”.

### P1.3 · El fixture de Puerto Umbral es inconsistente

- S03 y S03-mobile muestran Puerto Umbral como `Disponible`;
- S07 y S07-mobile la muestran `Parcial 5/8`;
- S05 muestra `Puerto Umbral · S01E03` en disco y visto;
- S07 muestra S01E03 `No descargado`;
- S10 anuncia S01E03 como siguiente episodio disponible.

Estas contradicciones invalidan la distinción central entre disponibilidad compartida y progreso personal.

### P1.4 · El player rompe el contexto de serie

El hotspot `Reproducir S01E01` de S07 abre S09, cuya captura y título corresponden a Órbita Ciega. Después S10 cambia de repente a Puerto Umbral S01E02. Hace falta una variante de player de episodio o un estado parametrizado que conserve título, episodio, progreso y retorno correctos.

### P1.5 · La suplantación no es persistente

S13 → `Ver como Bruno` navega a S03 y muestra un toast que afirma que la banda persiste, pero la captura S03 no contiene ninguna banda. La indicación aparece sólo al entrar en S12. El requisito exige que sea global, visible en todas las páginas y overlays.

### P1.6 · El recorrido móvil se convierte silenciosamente en escritorio

Evidencia: `S03-mobile → S04 → S06`. Al no existir variante móvil de S04, `render()` pierde `mobilePreferred`; el siguiente destino tampoco vuelve a su variante móvil. Esto impide cumplir los recorridos críticos móviles y puede ocultar problemas de layout reales.

### P1.7 · La carcasa móvil cambia entre pantallas

S03-mobile usa tres destinos y un panel `Estado`; S06-mobile introduce `Servidores`; S07-mobile muestra seis destinos, incluidos Plarr, Jackett y qBittorrent. La arquitectura global deja de ser predecible y S07 dedica demasiado ancho a integraciones. Debe existir una sola navegación primaria y un patrón secundario uniforme para salud.

### P1.8 · Faltan estados degradados accionables

qBittorrent aparece degradado en S03/S13, pero no hay una versión del detalle que demuestre la ausencia de Torrent; tampoco una película con Plarr caído que conserve metadata/torrent pero retire Play/Download. La matriz de la sección 9 no puede validarse con navegación o conmutadores.

## Hallazgos P2

### P2.1 · El selector no siempre respeta la opción elegida

Después de entrar en una captura móvil, seleccionar directamente su equivalente de escritorio vuelve a resolverlo hacia móvil mientras `mobilePreferred` siga activo. Para salir hay que usar `Vista escritorio` o pasar por una pantalla sin variante. El selector debe ser autoritativo.

### P2.2 · Áreas táctiles por debajo del mínimo

Los hotspots tienen `min-width/min-height: 2.5rem`, que resultó en 40×40 px en 320/390/tablet. El contrato pide aproximadamente 44 px. El cierre de S11-mobile y varias acciones de fila quedan en 40 px.

### P2.3 · Hotspots móviles imprecisos

En S11-mobile, `Enviar resultado` está en `y=52.2%`, aproximadamente sobre la primera fila que ya dice `Enviado`, no sobre la segunda acción `Enviar`. Esto hace que el hotspot y el estado visual discrepen. También `Comprobar biblioteca` en S11 de escritorio ocupa una franja amplia próxima al banner, no una acción visual inequívoca.

### P2.4 · El contenido estático no es accesible semánticamente

Los 96 hotspots son botones reales y tienen `aria-label`, pero todos los títulos, estados, rutas, progreso y mensajes dentro de la PNG quedan reducidos a un alt genérico. Un lector de pantalla conoce las acciones, no la información necesaria para decidir. Añadir una transcripción/inspector semántico por pantalla resolvería el límite propio del formato de captura.

### P2.5 · El panel de flujo domina la vista estrecha

A 320 px, toolbar + descripción + recorrido guiado ocupan casi toda la primera pantalla; el mockup empieza cerca del final del viewport y exige un scroll largo antes de poder interactuar. El recorrido debe poder contraerse o convertirse en barra de progreso compacta.

### P2.6 · El flujo activo queda obsoleto al navegar libremente

Tras elegir un recorrido, cambiar manualmente de pantalla conserva su título y sus pasos aunque la pantalla no pertenezca al flujo. En la prueba, S11-mobile seguía mostrando `Conexión lenta`. Debe limpiarse el recorrido, o reubicar el índice sólo cuando la pantalla pertenezca a él y marcar explícitamente “fuera del recorrido”.

### P2.7 · No existe variante móvil de bootstrap

La checklist exige bootstrap/login móvil, pero sólo existe S02-mobile. La primera ejecución no puede validarse a 320/390 ni con teclado virtual.

## Hallazgos P3

### P3.1 · Adelante nunca se deshabilita

Atrás/Adelante funcionan, pero `forward` no actualiza su estado y permanece habilitado incluso al final del historial.

### P3.2 · El panel Información no se comporta como diálogo

Tiene nombre y Escape/cierre, pero no mueve foco al abrir, no declara `role=dialog` y no atrapa foco. Es deuda del contenedor de revisión, no de la UI de Nestrr.

### P3.3 · S00 ofrece tres hotspots sin affordance visual

`Abrir bootstrap`, `Abrir login` y `Continuar con sesión válida` son tres rectángulos pequeños sobre el spinner; sólo se comprenden con zonas visibles y tooltip. Para una demo, conviene un selector explícito de condición de sesión fuera de la captura.

## Deudas reales del producto, separadas del prototipo

No se contabilizan como defectos del prototipo los puntos que el propio Markdown ya declara como deuda de producto: payload insuficiente de Seguir viendo para episodios, acción manual de visto no expuesta en la SPA actual, foco de episodio tras autoplay fallido, error parcial de temporada, rutas desconocidas y diálogos admin todavía basados en `prompt/confirm`. El rediseño debe reservarlos, pero su corrección exige implementación/backend además de capturas.

## Cambios concretos recomendados

1. Crear un pequeño modelo de estado por pantalla y capturas/variantes separadas: por ejemplo `S02-ready/loading/error`, `S04-idle/loading/results/empty/error`, `S11-ready/sending/accepted/rejected` y `S13-list/create/reset/confirm-delete`.
2. Definir un único fixture coherente en JSON y derivar de él todas las capturas: disponibilidad de título/episodio, progreso, salud y rutas. Añadir una prueba de consistencia cruzada.
3. Rehacer el flujo de película con un título remoto distinto: detalle sin Play/Download → torrent aceptado → pantalla pendiente → biblioteca tras organizar → detalle disponible → player.
4. Separar las ramas episódicas: usar S10 sólo cuando el siguiente episodio está disponible; para siguiente ausente, saltar del player a S07 con el episodio enfocado y desde allí abrir S11.
5. Añadir un player de episodio de Puerto Umbral en escritorio y móvil; no reutilizar la captura de Órbita Ciega.
6. Aplicar la banda de suplantación como overlay del prototipo a todas las pantallas/overlays mientras el estado esté activo.
7. Añadir variantes accionables para Plarr caído, Jackett caído y qBittorrent caído, verificando qué botones desaparecen conforme a la matriz.
8. Completar las variantes móviles de cada paso crítico, al menos S01, S04, S05, S10, S12 y S13; mantener una sola carcasa y preservar el modo móvil durante todo el recorrido.
9. Hacer el selector de pantalla autoritativo, limpiar/sincronizar el flujo activo y deshabilitar correctamente historial.
10. Elevar hotspots a 44×44 px, recolocar los de S11-mobile y añadir pruebas visuales de hitbox sobre cada captura.
11. Añadir al panel Información una ficha semántica de la pantalla actual con estados, texto clave y relaciones; así el prototipo no depende sólo de una imagen.
12. Mostrar en S05 nombres físicos canónicos ingleses distintos de los títulos localizados, para probar de forma inequívoca el criterio de idioma.

## Veredicto final

**No aprobado todavía para implementar tal cual.** La navegación técnica es estable y la dirección visual merece conservarse, pero las contradicciones de datos y la falta de estados/recorridos móviles reales pueden trasladar decisiones incorrectas a producción. **Sí queda feedback relevante que justifica una segunda ronda** después de corregir, como mínimo, todos los P1 y los P2.1–P2.3.
