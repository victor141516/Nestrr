# Revisión UX/QA independiente · ronda 4

## Resumen

La ronda 4 focalizada cierra los tres hallazgos pendientes de la ronda 3. La suplantación vuelve a ser global también en Ajustes y ofrece una salida operativa; las variantes degradadas de Jackett y qBittorrent tienen fichas semánticas específicas; y los estados loading/sending, accepted/saved y error exponen respectivamente `aria-busy`, `role=status` y `role=alert`.

Resultado: **0 P0, 0 P1 y 0 P2 nuevos o pendientes**. Permanece únicamente el P3 ya conocido del contenedor de revisión —la preferencia móvil puede heredarse al iniciar otro recorrido—, que no afecta al diseño de Nestrr ni justifica otra iteración.

**Veredicto: aprobado para implementación. No queda feedback relevante que justifique una ronda 5.**

## Alcance y evidencia

- Leídos íntegramente `reviews/round-3.md` y los archivos actuales `prototype/app.js`, `prototype-data.js`, `index.html` y `styles.css`.
- Prototipo real probado en `https://irsouate.tailfbddd.ts.net:8445/prototype/?review=round4`, cargando los scripts y estilos `v=7`.
- `node docs/redesign/prototype/verify.mjs`: `OK: 85 pantallas, 193 hotspots, 7 recorridos.`
- Sin warnings ni errores en la consola del navegador durante la regresión.
- Regresión real del recorrido de película completada de extremo a extremo en móvil: `S03-mobile → S04-mobile → S06-remote-mobile → S11-ready-mobile → S11-loading-mobile → S11-mobile → S11-sending-mobile → S11-accepted-mobile → S06-pending-mobile → S05-mobile → S06-mobile → S09-mobile`.
- Regresión de accesibilidad repetida tanto en escritorio como en las seis variantes móviles equivalentes.

## Cierre de los hallazgos de ronda 3

| Hallazgo ronda 3 | Estado ronda 4 | Evidencia |
| --- | --- | --- |
| P1.1 La banda de suplantación desaparecía en Ajustes | **Cerrado** | La secuencia real `S13-member-menu → Ver como Bruno → S03 → Abrir ajustes → S12` conserva visible `#impersonation-overlay` y su botón `Volver al administrador`. Al activarlo termina en `S03`, oculta la banda y muestra “Has vuelto a la sesión de administrador.” |
| P2.1 Fichas degradadas heredaban la semántica sana | **Cerrado** | `S06-jackett-down` anuncia que Jackett no responde, mantiene reproducción/descarga, bloquea la consulta de índices y recupera únicamente Jackett. `S06-qbittorrent-down` anuncia que qBittorrent no responde, mantiene reproducción/descarga, bloquea el envío de un torrent nuevo y recupera únicamente qBittorrent. |
| P2.2 Loading/accepted/error sin contrato semántico completo | **Cerrado** | `S11-loading` y `S11-sending` exponen `aria-busy="true"`; `S11-accepted` y `S12-saved` anuncian la operación completada con `role=status`; `S11-error` y `S04-error` anuncian un error recuperable con `role=alert`. Las seis variantes móviles equivalentes ofrecen el mismo contrato. |

## Suplantación: secuencia reproducida

1. Se abrió `S13-member-menu` y se activó `Ver como Bruno`.
2. El prototipo navegó a `S03`; la banda y `Volver al administrador` eran visibles.
3. Se activó `Abrir ajustes`; el prototipo navegó a `S12` y la banda continuó visible.
4. Se activó `Volver al administrador`; el prototipo terminó en `S03`, ocultó la banda y confirmó la recuperación de la sesión administrativa.

La misma prueba en móvil llegó a `S12-menu-mobile` con la banda visible y volvió a `S03-mobile` con la banda oculta.

## Estados accesibles verificados

| Pantalla | `aria-busy` del marco | Rol del anuncio | Texto observado |
| --- | --- | --- | --- |
| `S11-loading` | `true` | `status` | “Torrent · cargando: operación en curso.” |
| `S11-sending` | `true` | `status` | “Torrent · enviando: operación en curso.” |
| `S11-accepted` | `false` | `status` | “Torrent · aceptado: operación completada.” |
| `S12-saved` | `false` | `status` | “Ajustes · guardado: operación completada.” |
| `S11-error` | `false` | `alert` | “Torrent · error: se produjo un error recuperable.” |
| `S04-error` | `false` | `alert` | “Buscar · error: se produjo un error recuperable.” |

Las variantes `-mobile` de las seis pantallas repitieron el mismo resultado.

## Hallazgos por prioridad

### P0

Ninguno.

### P1

Ninguno.

### P2

Ninguno.

### P3 conocido, no bloqueante

La preferencia de viewport móvil del contenedor puede mantenerse al iniciar un recorrido nuevo. En esta regresión, tras revisar la suplantación móvil, el flujo de película comenzó y terminó correctamente en sus variantes móviles. Es el mismo P3 documentado en la ronda 3: afecta al demostrador, no al producto diseñado, y no requiere otra ronda antes de implementar.

## Veredicto final

**Aprobado para implementación.** Los tres puntos que motivaron esta ronda están cerrados con evidencia funcional y semántica, la verificación estructural continúa pasando y la regresión no revela ningún problema P0, P1 o P2.

**No queda feedback relevante que justifique otra ronda de prototipo.** El siguiente paso adecuado es trasladar el diseño validado al frontend real y verificar allí comportamiento responsive, accesibilidad y flujos contra las APIs efectivas.
