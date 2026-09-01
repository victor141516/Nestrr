# Inventario de producto y flujos para el rediseño de Nestrr

## 1. Propósito y criterio de alcance

Este documento es el contrato funcional y de navegación que debe conservar un rediseño completo de la web de Nestrr. No propone un estilo visual concreto. Describe qué puede hacer la aplicación, quién puede hacerlo, qué información debe aparecer y qué estados necesita representar el prototipo navegable.

Fuentes auditadas:

- implementación actual de la SPA Vue (`web/src/App.vue`, `MediaCard.vue`, `api.ts`, `i18n.ts` y `style.css`);
- contratos HTTP y comportamiento del backend (`internal/server/server.go`, `docs/API.md` y pruebas);
- especificación de producto de `README.md` y sus cuatro mockups históricos;
- flujo E2E real (`e2e/runner/e2e.mjs`);
- instancia local saludable en `http://localhost:8080` el 28 de agosto de 2026.

Convenciones de alcance:

- **Actual**: existe y es accesible en la SPA actual.
- **Contrato**: está implementado en backend o exigido explícitamente por la especificación, pero puede estar incompleto o no expuesto en la SPA. Debe tenerse presente en el rediseño para no cerrarle el paso.
- **Fuera de alcance inicial**: no debe aparecer como si funcionase actualmente.

El rediseño debe mantener una distinción inequívoca entre:

1. disponibilidad compartida en el sistema de archivos;
2. progreso, estado visto y preferencias de reproducción, que son por usuario;
3. envío de una descarga a qBittorrent, que no equivale a contenido ya disponible;
4. salud de las integraciones externas.

## 2. Roles, contextos y permisos

### 2.1 Visitante de una instancia nueva

- No hay administrador todavía.
- Sólo ve el alta inicial.
- Puede reclamar la instancia creando el primer usuario; ese usuario se convierte en el único administrador.
- Debe ver claramente el riesgo: no existe token de bootstrap y la primera alta válida gana la carrera.
- Campos: usuario de 2 a 40 caracteres, contraseña de al menos 10 caracteres e idioma inicial inferido del navegador.

### 2.2 Visitante de una instancia configurada

- Sólo ve el inicio de sesión.
- No existe registro público.
- Introduce usuario y contraseña.
- El error de credenciales no revela cuál de ambos campos falló.

### 2.3 Miembro

- Puede usar inicio, búsqueda, biblioteca descargada, detalles, reproducción, descargas de archivo, torrents si están disponibles y ajustes propios.
- Ve sólo su progreso, estado visto, preferencias de audio/subtítulos y metadatos en su idioma.
- Comparte los mismos archivos y la misma disponibilidad con todos los demás usuarios.
- No puede administrar usuarios ni cambiar su propia contraseña en la versión inicial.

### 2.4 Administrador

- Tiene todas las capacidades de un miembro.
- Es el único administrador y debe estar identificado como tal.
- Puede crear, renombrar y borrar miembros; restablecer contraseñas; suplantar miembros; volver a su propia identidad; e inspeccionar configuración y estado del servidor.
- No puede convertir a otro usuario en administrador.
- No debe poder eliminar al administrador mediante el flujo normal.

### 2.5 Administrador suplantando a un miembro

- Navega y actúa con los datos, idioma y progreso del miembro.
- Conserva acceso a Administración únicamente para poder terminar la suplantación.
- Toda la web muestra un aviso persistente y muy visible con el nombre del usuario observado y la acción “Volver al administrador”.
- La suplantación y su finalización son auditables en backend.

### 2.6 Usuario de un enlace público de Plarr

- Está fuera de la SPA autenticada de Nestrr.
- Puede reproducir o descargar el archivo asociado si alcanza el host de Plarr.
- No recibe sesión, progreso ni navegación de catálogo de Nestrr cuando abre Plarr de forma independiente.
- La interfaz de Nestrr no debe sugerir que esos enlaces están protegidos por su autenticación.

## 3. Mapa de información

```text
Estado público
├─ Carga inicial
├─ Crear administrador (sólo primera ejecución)
└─ Iniciar sesión

Aplicación autenticada
├─ Cabecera global
│  ├─ Marca → Inicio
│  ├─ Inicio
│  ├─ Buscar
│  ├─ Descargado / Biblioteca
│  ├─ Estado de Plarr, Jackett y qBittorrent
│  └─ Menú de usuario
│     ├─ Ajustes
│     ├─ Administración (admin o suplantación)
│     └─ Cerrar sesión
├─ Inicio / descubrimiento
│  ├─ Destacado
│  ├─ Seguir viendo
│  └─ Popular ahora + filtro “sólo descargado”
├─ Buscar
│  ├─ Películas / series
│  ├─ Consulta de texto
│  ├─ Resultados + disponibilidad progresiva
│  └─ Filtro “sólo descargado”
├─ Descargado / biblioteca local
│  └─ Películas y episodios verificados en disco
├─ Detalle
│  ├─ Película
│  │  ├─ Reproducir / reanudar
│  │  ├─ Descargar original o transcodificado
│  │  ├─ Buscar torrent
│  │  ├─ Actualizar metadatos
│  │  ├─ Colección
│  │  └─ Recomendaciones
│  ├─ Serie
│  │  ├─ Disponibilidad agregada
│  │  ├─ Temporadas desplegables
│  │  ├─ Episodios: reproducir o buscar torrent
│  │  └─ Recomendaciones
│  └─ Colección
│     └─ Títulos de la saga → película
├─ Reproductor Plarr a pantalla completa
│  ├─ Cerrar
│  ├─ Controles, calidad y pistas dentro del iframe
│  ├─ Episodio anterior / siguiente
│  └─ Cuenta atrás de reproducción automática
├─ Modal de torrent
│  ├─ Editar consulta y buscar
│  ├─ Filtrar resultados
│  ├─ Editar categoría
│  └─ Enviar resultado a qBittorrent
├─ Ajustes personales
│  ├─ Idioma
│  └─ Borrar metadatos propios
└─ Administración
   ├─ Crear miembro
   ├─ Lista y acciones de usuarios
   └─ Estado/configuración del servidor
```

Rutas hash actuales:

| Ruta | Pantalla | Acceso |
| --- | --- | --- |
| `#/` y `#/home` | Inicio | Autenticado |
| `#/search` | Buscar | Autenticado |
| `#/library` | Descargado | Autenticado |
| `#/settings` | Ajustes | Autenticado |
| `#/admin` | Administración | Admin o admin suplantando |
| `#/movie/:tmdbId` | Detalle de película | Autenticado |
| `#/tv/:tmdbId` | Detalle de serie | Autenticado |
| `#/collection/:tmdbId` | Detalle de colección | Autenticado |

El reproductor y el buscador de torrents son overlays sobre la ruta actual, no rutas independientes. Las rutas desconocidas producen hoy una carcasa vacía; el rediseño debería reservar una pantalla de “no encontrado” o una vuelta segura a Inicio.

## 4. Elementos globales de la aplicación autenticada

### 4.1 Cabecera y navegación

Requisitos:

- marca Nestrr que vuelve a Inicio;
- accesos principales: Inicio, Buscar y Descargado;
- indicador de sección activa;
- avatar con inicial y menú de usuario;
- estado compacto de `plarr`, `jackett` y `qbittorrent` en escritorio;
- cada integración admite tres estados: saludable/configurada, configurada pero no alcanzable, no configurada;
- el detalle o tooltip del estado contiene nombre y mensaje seguro, nunca el error crudo del upstream;
- el menú contiene el nombre del usuario, Ajustes, Administración cuando corresponda y Cerrar sesión.

Estados:

- integración saludable;
- integración degradada;
- integración ausente;
- menú cerrado/abierto;
- sección activa;
- cabecera con banda de suplantación debajo;
- mensaje de error global superpuesto.

Nota actual: el menú se abre por `hover`, por lo que el rediseño debe convertirlo en una interacción accionable por toque y teclado sin perder ninguna opción.

### 4.2 Banda de suplantación

- Sólo aparece mientras el administrador actúa como otro usuario.
- Texto con el usuario efectivo.
- Acción inequívoca para volver al administrador.
- Se mantiene visible al navegar entre todas las pantallas y overlays.

### 4.3 Notificaciones y errores

- Toast breve para guardado, limpieza de metadatos, metadatos actualizados, torrent aceptado, contraseña restablecida, contenido marcado como visto y fin de serie.
- Mensaje global para fallos de carga de ruta, catálogo o estado.
- Error localizado dentro del formulario de autenticación.
- Éxito o error persistente dentro del modal de torrent, además del toast.
- Las operaciones destructivas requieren confirmación explícita.
- Los botones ocupados se deshabilitan y muestran estado en curso.

### 4.4 Sesión

- Access token de dos horas almacenado sólo en memoria.
- Refresh token rotatorio en cookie `HttpOnly`.
- Renovación preventiva aproximadamente un minuto antes de caducar mientras la pestaña es visible.
- Al recuperar foco/visibilidad se comprueba la caducidad.
- Ante `401`, una renovación reactiva y un único reintento.
- Si el refresh está revocado o expirado, se limpia la sesión y se vuelve al login.
- No se exige todavía coordinación entre pestañas o dispositivos.

## 5. Catálogo de pantallas y requisitos

### S00. Carga inicial

**Objetivo:** resolver si la instancia necesita bootstrap y si existe una sesión renovable.

- Pantalla completa, sin navegación ni contenido intermedio.
- Indicador de actividad.
- Al terminar: S01, S02 o S03.
- Estado de error de red/servidor con reintento; hoy puede quedarse sólo el spinner o mostrar el login sin explicación si falla la inicialización.

### S01. Crear administrador

**Origen:** primera visita cuando `/setup.needsBootstrap=true`.

**Contenido obligatorio:**

- marca y promesa breve del producto;
- etiqueta “Primera ejecución”;
- título “Crear el administrador”;
- advertencia sobre la carrera de primera cuenta;
- usuario, contraseña y acción principal;
- reglas mínimas visibles o asociadas a los campos;
- estado enviando y error de validación/conflicto.

**Resultado:** sesión iniciada como administrador e Inicio.

**Errores relevantes:** usuario inválido, contraseña menor de 10 caracteres, administrador creado por otra petición (`bootstrap_unavailable`), JSON/red/servidor.

### S02. Iniciar sesión

**Origen:** visitante sin sesión cuando ya existe administrador.

**Contenido obligatorio:** usuario, contraseña, acción “Iniciar sesión”, estado enviando y error de credenciales.

**Resultado:** Inicio en el idioma del usuario.

No incluye registro, recuperación de contraseña ni cambio de contraseña propio en el alcance inicial.

### S03. Inicio y descubrimiento

**Datos:** dos descubrimientos TMDB en paralelo (películas y series, popularidad descendente), hasta 12 de cada tipo; progreso del usuario; disponibilidad emitida progresivamente.

**Bloques:**

1. **Destacado**: primer elemento combinado, backdrop, título localizado, sinopsis, valoración y acción “Ver detalles”.
2. **Seguir viendo**: aparece sólo si existe progreso; hasta ocho entradas, tipo/identidad, barra calculada con posición/duración y navegación al contenido.
3. **Popular ahora**: rejilla mixta de películas y series con título, año, nota, póster y badge local/parcial/remoto.
4. **Sólo descargado**: filtra en cliente los resultados ya cargados; incluye películas disponibles y series parciales o completas.

**Estados visuales:** carga inicial, hero con/sin imagen, contenido sin sinopsis/nota/año, disponibilidad aún desconocida, disponible, parcial, no disponible, sin resultados, progreso vacío y error de catálogo.

**Incidencia a resolver en el diseño/implementación posterior:** las tarjetas actuales de Seguir viendo sólo enseñan `tipo #id`; para episodios no guardan en la respuesta el `showId`, por lo que su destino puede ser incorrecto. El rediseño debe representar el dato humano esperado (título, episodio, imagen, posición y acción reanudar), pero requiere confirmar/ampliar el payload.

### S04. Buscar

**Contenido:**

- título/introducción “Explorar TMDB”;
- selector mutuamente excluyente Películas/Series;
- consulta libre editable;
- acción Buscar;
- filtro “Sólo descargado”;
- rejilla con las mismas tarjetas y estados de disponibilidad que Inicio.

**Comportamiento:**

- una consulta vacía no envía petición;
- la búsqueda usa el idioma del usuario para los metadatos visibles;
- al terminar, sustituye los resultados previos;
- las comprobaciones de disponibilidad llegan progresivamente;
- al cambiar el filtro no se relanza TMDB;
- pulsar una tarjeta abre su detalle.

**Estados:** antes de la primera búsqueda, buscando, resultados, cero resultados, cero resultados por el filtro, error y resultados con badges aún pendientes.

**Contrato disponible pero no expuesto hoy:** TMDB soporta listas de géneros y filtros de descubrimiento. No se deben simular como feature activa salvo decisión explícita, pero la arquitectura visual no debería impedir filtros de catálogo posteriores.

### S05. Descargado / biblioteca local

**Objetivo:** enumerar directamente el sistema de archivos organizado, que es la autoridad.

**Contenido por fila:**

- marca de disponibilidad verificada;
- póster si la resolución TMDB tuvo éxito;
- título localizado si existe, o título canónico derivado del archivo;
- película: año;
- episodio: código `SxxExx`;
- ruta canónica;
- fila accionable sólo si existe identidad TMDB resoluble.

**Comportamiento:** al abrir, vuelve a enumerar la biblioteca; los nuevos hardlinks de Chiprr deben aparecer sin rescan global. Películas llevan a detalle de película y episodios a la serie resuelta.

**Estados:** cargando, vacía, mezcla de películas/episodios, elemento con metadatos, elemento sin metadatos/no accionable, imagen ausente y error de enumeración.

El nombre “Descargado” significa “presente en la biblioteca organizada”, no “tarea de qBittorrent terminada”. No existe UI de cola o progreso de torrent en la versión inicial.

### S06. Detalle de película

**Datos dinámicos:** documento TMDB localizado, disponibilidad canónica inglesa, progreso/preferencias del usuario, salud de integraciones, colección y hasta 12 recomendaciones.

**Hero obligatorio:** backdrop, póster, tipo, año, título, nota, duración si existe, hasta tres géneros, sinopsis y estado local.

**Acciones condicionadas:**

- **Reproducir/Reanudar**: sólo si el archivo está disponible, hay URL de Plarr y Plarr está saludable;
- **Descargar**: sólo si existe archivo, URL de descarga y Plarr saludable;
- **Buscar torrents**: sólo si Jackett y qBittorrent están saludables;
- **Actualizar metadatos**: siempre para el usuario autenticado;
- **Marcar visto/no visto**: exigido por el contrato de producto y endpoint, aunque la SPA actual no lo muestra.

**Menú de descarga:** Original y rendiciones MP4 `1080p`, `720p`, `480p`, `360p`, `240p`. Plarr no debe precomputar el archivo completo antes de empezar a entregarlo. El diseño debe admitir tamaños/descripciones si en el futuro están disponibles, sin inventarlos ahora.

**Secciones inferiores:**

- colección/saga cuando `belongs_to_collection` existe;
- relacionados y recomendados, con tarjetas y disponibilidad progresiva.

**Estados:** disponible/no disponible/comprobando; Plarr sano/caído/ausente; torrents disponibles/no disponibles; sin póster/backdrop/sinopsis/nota/géneros; sin colección; sin recomendaciones; actualización en curso/éxito/error; estado visto/progreso/reanudar.

**Regla crítica de idioma:** la UI y los metadatos visibles siguen el idioma del usuario, pero la comprobación de ruta siempre usa el título inglés canónico de TMDB. El idioma configurado sirve además como preferencia inicial de pista de audio si todavía no existe una selección guardada.

### S07. Detalle de serie y temporadas

**Hero:** backdrop, póster, tipo “Serie”, año o intervalo si TMDB lo aporta, título, nota, géneros, sinopsis y disponibilidad agregada.

**Disponibilidad agregada:**

- completa si todos los episodios soportados están disponibles;
- parcial si hay al menos uno pero no todos;
- ninguna si no hay episodios disponibles.

Debe mantenerse separada del porcentaje visto por el usuario. Una serie puede estar completa en disco y no vista, o parcialmente descargada y muy avanzada en visionado.

**Temporadas:**

- excluir temporada 0/especiales en el alcance inicial;
- lista desplegable por temporada con póster, nombre, cantidad de episodios, año, estado completa/parcial/ninguna y contador descargado/total;
- cada temporada carga sus episodios TMDB.

**Episodios:** número, título, imagen fija, sinopsis, disponibilidad, valoración/duración si se exponen, progreso/visto por usuario y acciones:

- Reproducir/Reanudar sólo con archivo y Plarr saludable;
- Buscar torrent si Jackett y qBittorrent están saludables, esté o no disponible;
- refrescar o marcar visto son capacidades de contrato a contemplar aunque no estén expuestas por episodio en la SPA actual.

**Estados:** carga incremental de temporadas y disponibilidad; temporada cerrada/abierta; episodios sin imagen/sinopsis; ninguno disponible; parcial; completa; episodio visto/no visto/con progreso; integración degradada; error de una temporada sin perder el resto.

### S08. Detalle de colección/saga

**Datos:** artwork y descripción de colección si TMDB los ofrece, lista `parts` de películas.

**Contenido:** hero reutilizado con tipo “Colección”, título, año si existe, sinopsis, backdrop/póster y sección “Títulos” con tarjetas de películas.

**Navegación:** cada título abre S06.

**Acciones que no corresponden:** reproducir la colección, disponibilidad binaria o buscar un torrent para toda la colección.

**Estados:** colección completa, sin artwork, sin descripción, sin partes, partes sin disponibilidad todavía y error.

### S09. Overlay del reproductor Plarr

**Entrada:** acción Reproducir/Reanudar de una película o episodio.

**Contenedor Nestrr:**

- ocupa toda la ventana sobre la ruta de detalle;
- botón cerrar siempre accesible;
- iframe de Plarr con `autoplay`, fullscreen y picture-in-picture permitidos;
- valida origen, ventana, versión y forma de cada `postMessage`;
- persiste eventos en nombre del usuario autenticado.

**Parámetros al abrir:**

- `start`: cinco segundos antes de la última posición guardada;
- `audio`: pista guardada o, si no existe, subtag primario del idioma del usuario;
- `subtitles`: idioma guardado;
- `subtitlesEnabled=true` cuando correspondía;
- `parentOrigin`: origen de Nestrr;
- la calidad siempre empieza en Auto, no se restaura entre sesiones.

**Capacidades dentro de Plarr que la captura debe comunicar:** play/pausa, timeline, buffering, volumen, pantalla completa/PiP, calidad Auto/manual, audio, subtítulos, error de reproducción y anterior/siguiente cuando sea un episodio. Plarr puede hacer direct play/remux o transcodificar a H.264 y adaptar calidad conservadoramente, incluyendo rendiciones muy bajas para conexiones malas.

**Mensajes relevantes:** opened/ready, play, pause, seek, heartbeat cada cinco segundos, ended, closed/page-hidden, cambios de audio/subtítulos/calidad, anterior/siguiente y playback-error.

**Cierre y final:**

- `closed` o botón × cierra el overlay y cancela autoplay;
- `ended` marca visto y muestra confirmación;
- en una película termina ahí;
- en episodio empieza S10.

### S10. Cuenta atrás y navegación episódica

- Al finalizar un episodio aparece una cuenta atrás de 5 segundos.
- Acción Cancelar conserva la pantalla del reproductor sin saltar automáticamente.
- Al llegar a cero, pide el episodio siguiente.
- Anterior/siguiente atraviesa límites de temporada.
- Si existe y está disponible, reemplaza el iframe por el nuevo episodio.
- Si existe pero no está disponible, cierra el reproductor, navega al detalle de la serie/episodio y muestra “No descargado”; desde allí se puede buscar torrent.
- Si no existe otro episodio, cierra el reproductor y muestra “Fin de la serie”.
- El salto aceptado desde ≥95 % marca el episodio como visto; un seek arbitrario no.

### S11. Modal de búsqueda y envío de torrent

**Disponibilidad:** accesible sólo cuando Jackett y qBittorrent están ambos saludables. Se abre desde película, serie o episodio sin abandonar el detalle.

**Cabecera/contexto:** título de la acción, origen Jackett → destino qBittorrent, contenido para el que se abrió y cierre.

**Consulta inicial y categoría:**

- película: título inglés/original de la película;
- serie o episodio: título inglés/original de la serie, sin añadir automáticamente temporada/episodio;
- consulta completamente editable;
- categoría prellenada con la de películas o series configurada por el servidor;
- categoría libremente editable y no cambia al editar la consulta;
- se permite deliberadamente una categoría “incongruente” con el contenido.

**Búsqueda:**

- botón Buscar y estado “Buscando…”;
- error seguro si Jackett falla;
- resultado admite magnet o `.torrent` normalizado por backend.

**Filtros locales, sin nueva petición:**

- mínimo de semillas;
- tracker múltiple;
- seleccionar todos / limpiar todos;
- varias palabras obligatorias separadas por coma o espacio;
- varias palabras excluidas;
- orden descendente por semillas.

**Fila de resultado:** título completo, tracker, semillas, tamaño convertido a GB y acción Enviar.

**Estados de envío:**

- listo;
- enviando: deshabilitar el resto de envíos y mostrar “Enviando…” en la fila;
- aceptado: fila en “Enviado”, mensaje persistente `role=status` y toast de confirmación;
- error: mensaje persistente `role=alert`, botón vuelve a estar disponible y toast de fallo;
- qBittorrent gestiona duplicados; Nestrr no muestra cola ni porcentaje posterior.

**Estados vacíos que deben distinguirse:** aún no se buscó, búsqueda en curso, búsqueda sin resultados, filtros ocultan todos los resultados y fallo de búsqueda.

### S12. Ajustes personales

**Idioma:**

- opciones actuales: English (US), Español (España), Português y Français;
- al guardar, actualiza el usuario, confirma y vuelve a cargar la ruta en el nuevo idioma/metadatos;
- la interfaz sólo tiene traducciones completas inglesa y española; portugués/francés caen a inglés en UI, aunque TMDB sí recibe el idioma elegido;
- no renombra ni afecta rutas del sistema de archivos;
- sirve como pista de audio por defecto sólo hasta que el usuario elige otra para el contenido.

**Metadatos almacenados:**

- acción para borrar únicamente los documentos TMDB de larga duración del usuario;
- no borra archivos, imágenes compartidas, progreso ni datos de otros usuarios;
- necesita confirmación o copy suficientemente explícito antes de una operación destructiva;
- confirma al terminar.

**Estados:** guardando/guardado/error, borrado en curso/éxito/error y sesión suplantada.

### S13. Administración

**Acceso:** administrador real o administrador suplantando. Un miembro recibe `403` y no debe ver el acceso.

**Alta de usuario:** usuario, contraseña temporal (mínimo 10) e idioma; acción Añadir; tras éxito limpia el formulario y refresca la lista.

**Lista de usuarios:** avatar/inicial, nombre, rol, idioma, badge ADMIN y acciones por fila:

- Renombrar;
- Restablecer contraseña (revoca sesiones del usuario);
- Ver como usuario, sólo miembros;
- Eliminar, sólo miembros, con confirmación.

La implementación actual usa `prompt`/`confirm`; el rediseño debe representarlos como diálogos propios, con validación, cancelar, enviando, error y éxito.

**Estado del servidor:**

- TMDB configurado/ausente;
- host público de Plarr o deshabilitado;
- raíces de películas y series;
- cantidades de documentos de metadatos, assets de imagen y registros de progreso;
- el backend también entrega estados de integraciones, directorio de imágenes y categorías movie/show, que conviene poder presentar sin exponer secretos.

**Errores:** nombre duplicado, usuario/contraseña inválidos, usuario inexistente, intento de borrar admin, suplantación inválida, identidad de administrador perdida, acceso prohibido y error de servidor.

## 6. Componentes y datos dinámicos compartidos

### 6.1 Tarjeta de contenido

Datos:

- `id`, `media_type` (`movie` o `tv`), título/nombre localizado;
- póster, fecha de estreno/primera emisión, nota;
- disponibilidad `none`, `partial` o `complete`.

Estados de badge:

- `✓ Local/Descargado` para película disponible o serie completa;
- `◐ Parcial` para serie con algún episodio;
- `+`/sin descargar para remoto;
- pendiente mientras llega el stream de comprobación;
- imagen ausente o fallida.

Toda la tarjeta es accionable y abre su detalle. El título debe poder truncarse visualmente sin perder su nombre accesible completo.

### 6.2 Progreso

Datos persistentes por `(usuario, tipo, TMDB ID)`:

- posición y duración absolutas;
- fecha de actualización;
- visto y fecha;
- último evento/sesión Plarr;
- idioma y título de audio;
- idioma y título de subtítulo;
- subtítulos activos/inactivos.

Derivaciones visuales:

- porcentaje de progreso;
- Reanudar frente a Reproducir;
- Visto/no visto;
- progreso agregado de temporada/serie a partir de episodios.

La resolución/calidad nunca se persiste.

### 6.3 Disponibilidad

Se calcula a partir del nombre canónico inglés y del año/temporada/episodio, no de textos localizados. En listados llega como eventos SSE y puede cambiar tarjeta a tarjeta sin bloquear la pintura del catálogo.

Estados por item:

- comprobando/desconocida;
- ruta esperada pero archivo ausente;
- archivo presente;
- error de resolución del nombre canónico;
- serie parcial/completa/ninguna.

Cuando existe un archivo reproducible, se reciben `relativePath`, `playerUrl` y `originalDownloadUrl`. Una serie agregada nunca tiene un único `playerUrl`; se reproduce por episodio.

### 6.4 Metadatos e imágenes

- Textos y documentos TMDB están aislados por usuario, idioma, tipo e ID.
- Un usuario puede refrescar el item actual y borrar sus documentos.
- Imágenes se cachean en el servidor.
- El diseño debe ofrecer fallback ante póster/backdrop/still vacío o imagen no disponible.
- Cambiar idioma puede producir temporalmente mezcla de documentos hasta explorar, refrescar o limpiar metadatos; no debe presentarse como cambio de nombres físicos.

### 6.5 Integraciones

| Integración | Si está saludable | Si falta o está caída |
| --- | --- | --- |
| TMDB | catálogo, búsqueda, detalles, artwork | error/estado de configuración; la biblioteca puede conservar filas canónicas sin metadata |
| Plarr | play, original y transcodificados | ocultar reproducción y descargas; catálogo y torrents siguen funcionando |
| Jackett | búsqueda de torrents junto con qBittorrent | ocultar búsqueda de torrents |
| qBittorrent | envío de torrent junto con Jackett | ocultar búsqueda/envío de torrents |
| Chiprr | no tiene UI ni conexión directa | Nestrr sólo verá el archivo cuando aparezca en la biblioteca organizada |

## 7. Flujos de navegación y acciones

| # | Origen | Acción/condición | Destino o resultado |
| ---: | --- | --- | --- |
| 1 | S00 | Instancia sin admin | S01 |
| 2 | S00 | Instancia con admin y refresh válido | S03 |
| 3 | S00 | Instancia con admin y sin sesión | S02 |
| 4 | S01 | Crear admin válido | Sesión admin → S03 |
| 5 | S01 | Otra petición creó el admin primero | Error y paso seguro a S02 |
| 6 | S02 | Credenciales válidas | S03 |
| 7 | Cualquier pantalla | Token próximo a expirar | Refresh silencioso, misma pantalla |
| 8 | Cualquier pantalla | Refresh inválido/revocado | S02 |
| 9 | Cabecera | Logo o Inicio | S03 |
| 10 | Cabecera | Buscar | S04 |
| 11 | Cabecera | Descargado | S05 y nueva enumeración |
| 12 | Avatar | Ajustes | S12 |
| 13 | Avatar admin | Administración | S13 |
| 14 | Avatar | Cerrar sesión | Revocar sesión → S02 |
| 15 | S03 | Ver detalles del destacado | S06 o S07 |
| 16 | S03/S04 | Abrir tarjeta | S06 o S07 |
| 17 | S03 | Abrir entrada de Seguir viendo | Película S06 o serie/episodio S07 |
| 18 | S03/S04 | Activar Sólo descargado | Mismo screen, lista filtrada localmente |
| 19 | S04 | Elegir tipo + buscar | Resultados en S04 |
| 20 | S05 | Abrir fila con identidad TMDB | S06 o S07 |
| 21 | S05 | Fila no resuelta | Permanece; no es accionable |
| 22 | S06 | Abrir colección | S08 |
| 23 | S08 | Abrir parte | S06 |
| 24 | S06/S07 | Abrir recomendación | S06 o S07 del recomendado |
| 25 | S06 | Reproducir/Reanudar | S09 con progreso y preferencias |
| 26 | S07 | Desplegar temporada | Lista de episodios en el mismo S07 |
| 27 | S07 | Reproducir/Reanudar episodio | S09 |
| 28 | S09 | Cerrar o evento closed | Vuelve al detalle subyacente |
| 29 | S09 episodio | Evento ended | Toast visto + S10 |
| 30 | S10 | Cancelar | Sigue en S09 sin autoplay |
| 31 | S10 | Cuenta llega a cero; siguiente disponible | S09 reemplaza iframe |
| 32 | S09 | Anterior/siguiente disponible | S09 reemplaza iframe |
| 33 | S09/S10 | Episodio adyacente no disponible | Cierra player → S07 + aviso; torrent si posible |
| 34 | S09/S10 | Fin de serie | Cierra player → S07 + toast |
| 35 | S06 | Descargar Original | Descarga pública Plarr |
| 36 | S06 | Elegir rendition | Descarga MP4 transcodificada en streaming |
| 37 | S06/S07 | Buscar torrent | S11 sobre el detalle |
| 38 | S11 | Editar consulta y Buscar | Resultados Jackett en S11 |
| 39 | S11 | Cambiar filtros | Misma lista filtrada, sin red |
| 40 | S11 | Enviar un resultado | Enviando → aceptado/error en S11 |
| 41 | S11 | Aceptado | Toast; la película aún sigue “no descargada” |
| 42 | Externo | qBit termina + Chiprr organiza | Una nueva enumeración/comprobación la marca disponible |
| 43 | S06/S07/S08 | Actualizar metadatos | Recarga item, confirmación, misma ruta |
| 44 | S12 | Guardar idioma | Sesión actualizada, misma ruta recargada |
| 45 | S12 | Borrar metadatos propios | Confirmación/éxito, misma pantalla |
| 46 | S13 | Crear miembro | Lista actualizada |
| 47 | S13 | Renombrar miembro | Diálogo → lista actualizada |
| 48 | S13 | Resetear contraseña | Diálogo → sesiones revocadas + toast |
| 49 | S13 | Eliminar miembro | Confirmación → lista actualizada |
| 50 | S13 | Ver como usuario | Sesión suplantada → S03 + banda persistente |
| 51 | Banda suplantación | Volver al administrador | Sesión admin → S03 |
| 52 | Ruta desconocida | Navegar/cargar | Estado 404 con acción Inicio (mejora necesaria) |

## 8. Flujos detallados críticos

### 8.1 Descubrir → descargar → organizar → reproducir

1. El usuario encuentra un item TMDB en Inicio o Buscar.
2. La comprobación de biblioteca lo marca como no descargado.
3. Abre detalle y, si Jackett/qBittorrent están sanos, abre S11.
4. La consulta inicial usa el nombre inglés, pero el usuario puede reescribirla.
5. Filtra y envía un resultado con la categoría deseada.
6. Nestrr confirma únicamente “aceptado por qBittorrent”; no simula avance.
7. qBittorrent descarga fuera de la biblioteca y mueve el terminado al directorio completed.
8. Chiprr reconoce el archivo y crea el hardlink canónico.
9. Al volver a abrir el item o S05, Nestrr revalida el sistema de archivos y muestra disponibilidad.
10. Si Plarr está sano, aparece Reproducir y Descargar.
11. El usuario abre S09 y el vídeo real avanza.

Estados visuales indispensables: no descargado, búsqueda, resultado, enviando, aceptado, todavía no disponible, recién disponible, player cargando, reproduciendo y error de playback.

### 8.2 Reanudar con conexión lenta

1. Nestrr abre Plarr con `start` y preferencias guardadas.
2. Plarr empieza en Auto y puede elegir la rendition mínima; debe existir representación de buffering.
3. Debe bajar calidad rápidamente ante riesgo de corte y subir con cautela.
4. Cambios de rendimiento no recargan la página ni mueven el timeline.
5. El usuario puede fijar calidad manual y volver a Auto.
6. Heartbeats actualizan la posición aproximadamente cada cinco segundos.
7. Al cerrar, el próximo inicio reanuda cinco segundos antes; la calidad vuelve a Auto.

### 8.3 Serie parcialmente descargada

1. La tarjeta de serie muestra Parcial, no “Descargada” sin matiz.
2. El hero separa disponibilidad agregada de progreso visto.
3. Cada temporada muestra `disponibles/total`.
4. Cada episodio muestra su propio estado y progreso.
5. Episodio local + Plarr sano: reproducir/reanudar.
6. Episodio ausente + integraciones de torrent sanas: buscar torrent.
7. Un siguiente episodio ausente interrumpe autoplay y aterriza en S07.

### 8.4 Operación degradada

- Sin Plarr: catálogo, detalles, biblioteca, progreso ya guardado, usuarios y torrents continúan; no hay Play ni descargas de archivo.
- Sin Jackett o sin qBittorrent: catálogo, biblioteca, Plarr y ajustes continúan; no hay controles de torrent.
- Sin TMDB: se muestra error/configuración; filas físicas no resueltas pueden seguir existiendo en biblioteca.
- Chiprr no se representa como integración en vivo, porque Nestrr no se comunica con él.

## 9. Matriz de acciones por estado

| Estado del contenido/servicio | Play | Descargar archivo | Buscar torrent | Refrescar metadata |
| --- | :---: | :---: | :---: | :---: |
| Archivo presente + Plarr sano | Sí | Sí | Sí si Jackett+qBit sanos | Sí |
| Archivo presente + Plarr caído | No | No | Sí si Jackett+qBit sanos | Sí |
| Archivo ausente | No | No | Sí si Jackett+qBit sanos | Sí |
| Jackett caído o ausente | Según Plarr | Según Plarr | No | Sí |
| qBittorrent caído o ausente | Según Plarr | Según Plarr | No | Sí |
| Colección | No | No | No | Sí |
| Serie agregada | No directo; por episodio | No directo; por episodio | Sí desde contexto/episodio | Sí |

## 10. Responsive y móvil

Breakpoint actual: `760px`. La cobertura de rediseño debe incluir al menos 320 px, un móvil típico de 390 px, tablet y escritorio ancho.

Comportamiento actual que debe conservarse funcionalmente, aunque cambie el patrón:

- login/bootstrap pasa de dos columnas a una; se oculta el bloque editorial y queda el formulario;
- cabecera reduce marca a isotipo, oculta estados de integración y compacta navegación;
- hero baja de altura y reposiciona el backdrop;
- rejillas pasan a dos columnas;
- detalle oculta póster y apoya el contenido sobre el backdrop;
- formularios admin y filtros torrent pasan a una columna;
- searchbar puede envolver líneas;
- lista de episodios reduce miniaturas y elimina sangría;
- modal torrent conserva scroll interno y no puede superar la ventana;
- player permanece completamente fullscreen;
- toast, errores globales y autoplay no deben quedar fuera de viewport ni tapar controles críticos.

Requisitos móviles que el rediseño debe resolver explícitamente:

- navegación principal accesible sin depender de hover;
- menú de usuario por toque con cierre predecible;
- estados de integración accesibles desde un panel/menú aunque se oculten los puntos;
- todas las acciones de detalle disponibles sin desbordamiento horizontal;
- menú de descarga usable por toque;
- filas de episodio apilables sin perder disponibilidad, progreso ni acciones;
- filtros de torrent editables y resultados legibles; título largo debe envolver, no empujar Enviar fuera de pantalla;
- confirmaciones, prompts admin y formularios con teclado virtual abierto;
- áreas táctiles de al menos ~44 px;
- respeto por safe areas en player, modales y navegación;
- estados de foco visibles y orden de tabulación lógico en teclado/tablet;
- fallback para orientación horizontal y poco alto, especialmente player y modal torrent.

## 11. Estados de carga, vacíos, errores y confirmación

### 11.1 Carga

- inicialización completa;
- login/bootstrap enviando;
- carga de ruta/catálogo;
- búsqueda TMDB;
- disponibilidad individual en streaming;
- biblioteca enumerando;
- detalle y temporadas;
- actualización de metadatos;
- búsqueda Jackett;
- envío qBittorrent;
- ajustes/admin guardando;
- player/manifest/transcoding/buffering;
- descarga transcodificada empezando y continuando en stream.

### 11.2 Vacíos

- Inicio sin resultados TMDB;
- Seguir viendo sin progreso (sección oculta);
- búsqueda aún no ejecutada;
- búsqueda sin coincidencias;
- resultados existentes pero cero tras “Sólo descargado”;
- biblioteca física vacía;
- archivo sin identidad/metadata TMDB;
- detalle sin sinopsis, artwork, géneros, valoración, colección o recomendaciones;
- colección sin partes;
- serie sin temporadas compatibles;
- temporada sin episodios;
- torrent antes de buscar;
- Jackett sin resultados;
- filtros torrent ocultan todo;
- admin sin miembros aparte del administrador.

### 11.3 Errores

- setup o servidor inaccesible;
- autenticación inválida, sesión expirada y origen de refresh rechazado;
- catálogo/TMDB fallido o ruta desconocida;
- artwork no cacheable;
- disponibilidad no resoluble en inglés;
- biblioteca no enumerada;
- Plarr inalcanzable o playback unavailable;
- error de navegación a episodio adyacente;
- Jackett no responde;
- qBittorrent rechaza el torrent;
- validaciones y conflictos de usuario;
- pérdida de conexión durante una mutación.

Los errores deben ser accionables y seguros: reintentar/cerrar/volver, conservar entradas cuando sea posible y no mostrar stack traces, secretos ni respuestas upstream crudas.

### 11.4 Éxitos/confirmaciones

- sesión iniciada;
- ajustes guardados;
- metadata actualizada o borrada;
- torrent aceptado;
- usuario creado/renombrado/eliminado;
- contraseña restablecida y sesiones revocadas;
- suplantación iniciada/finalizada;
- item visto;
- descarga de archivo iniciada.

## 12. Idioma, contenido y nombres físicos

- Cada usuario elige su idioma; no hay ajuste global del administrador.
- UI: inglés y español; cualquier otro idioma cae a inglés.
- TMDB: se consulta con la etiqueta BCP 47 elegida, incluyendo portugués o francés.
- Sistema de archivos: siempre título inglés canónico, sin depender del idioma de UI.
- Torrent: consulta inicial en inglés/original, siempre editable.
- Audio: el subtag primario configurado (`es` para `es-ES`) se usa como preferencia inicial si no hay selección guardada.
- Preferencias explícitas de pista prevalecen sobre el idioma del usuario.
- Textos hardcodeados que hoy permanecen en inglés aun con UI española (eyebrows, filtros, admin, disponibilidad parcial, etc.) deben entrar en el sistema de traducción en el rediseño.

## 13. Accesibilidad y semántica mínima

- Todas las acciones deben ser botones o enlaces reales; las tarjetas y filas accionables necesitan equivalente por teclado.
- Foco visible, orden lógico y restauración de foco al cerrar modal/player/diálogo.
- Modal torrent y diálogos admin deben atrapar foco, tener nombre accesible y cerrar con Escape sin descartar una operación ya enviada.
- Mensaje torrent exitoso usa estado no intrusivo; error usa alerta.
- Loading indica `aria-busy`; resultados dinámicos anuncian el cambio sin leer toda la rejilla.
- Badges no dependen sólo del color: texto/símbolo para disponible, parcial y ausente.
- Integraciones no dependen sólo del punto rojo/verde.
- Imágenes decorativas con `alt` vacío; posters informativos con nombre útil o texto vecino suficiente.
- Contraste usable sobre backdrops mediante overlays consistentes.
- Soportar `prefers-reduced-motion` para transiciones, spinners no esenciales y autoplay visual; no desactivar el temporizador funcional sin una decisión explícita.
- Cuenta atrás de siguiente episodio debe anunciarse sin bombardear al lector cada segundo y siempre ofrecer Cancelar.

## 14. Límites funcionales que no deben representarse como existentes

- No hay cola, porcentaje, pausa, reanudación ni borrado de torrents dentro de Nestrr.
- No hay comunicación directa ni botón “rescanear Chiprr”.
- No hay múltiples archivos por película/episodio, multi-episode, anime absoluto, especiales/temporada cero ni sidecars iniciales.
- No hay ranking personalizado, “Mi lista”, historial dedicado, calendario o actividad como features implementadas, aunque algunos mockups históricos los dibujen.
- No hay Chromecast ni AirPlay iniciales.
- No hay registro público, recuperación de contraseña ni cambio de contraseña propio para miembros.
- No hay selector persistente de calidad; cada sesión empieza en Auto.
- No hay reproducción única de una serie o colección: se reproduce un episodio/archivo.
- No hay protección Nestrr sobre enlaces públicos de Plarr.
- No hay interfaz de configuración de secretos, rutas o credenciales; admin sólo inspecciona valores seguros.

## 15. Deudas o ambigüedades detectadas que afectan al prototipo

Estas preguntas no impiden dibujar el flujo, pero deben quedar explícitas para evitar fijar un comportamiento accidental:

1. **Seguir viendo episodios:** el payload actual no ofrece nombre/imagen/show ID suficiente y la ruta usa el ID del episodio como ID de serie. Se necesita contrato ampliado para una tarjeta útil y destino correcto.
2. **Marcado manual de visto:** backend y especificación lo soportan, pero la SPA no expone el botón. El rediseño debería incluir el estado/acción y validar su ubicación.
3. **Progreso en detalles:** se usa para abrir Plarr, pero la pantalla actual no presenta posición, visto o preferencias. Debe visualizarse sin confundirlo con disponibilidad.
4. **Estado desconocido de disponibilidad:** la tarjeta actual parece remota (`+`) antes de recibir SSE; conviene distinguir “comprobando” de “no descargado”.
5. **Error parcial por temporada:** la carga actual es secuencial y un fallo aborta el detalle completo. El prototipo debería reservar un estado de reintento por sección.
6. **Detalle focal de episodio:** `?season=&episode=` se genera al fallar autoplay, pero hoy no despliega ni enfoca automáticamente la temporada/episodio. El flujo esperado necesita ese foco.
7. **Administración en móvil:** los botones por usuario y `prompt/confirm` no tienen un patrón robusto; se requieren diálogos y menú de acciones.
8. **Integraciones en móvil:** hoy se ocultan por completo. Debe existir una vía secundaria de consulta.
9. **Rutas inválidas:** hoy dejan una pantalla vacía. Falta 404/retorno seguro.
10. **Idioma:** varias cadenas de la SPA están hardcodeadas en inglés. El rediseño visual debe prever expansión de texto y cobertura completa.
11. **Serie y torrent:** la SPA sólo abre torrent desde episodios, no desde el hero general de la serie; el contrato de consulta habla de episodio/show. Hay que confirmar si se quiere una acción a nivel serie además de por episodio.
12. **Biblioteca no resuelta:** una fila física sin match TMDB no lleva a detalle. Hace falta decidir si sólo se informa o se ofrece diagnóstico, sin inventar gestión de archivos.

## 16. Checklist de cobertura visual

El prototipo de capturas y áreas clicables no está completo hasta representar y enlazar al menos lo siguiente.

### Autenticación y sesión

- [ ] S00 carga inicial.
- [ ] S01 bootstrap normal, validación y conflicto “admin ya creado”.
- [ ] S02 login normal, enviando y credenciales inválidas.
- [ ] Expiración definitiva de sesión → login.
- [ ] Escritorio y móvil para bootstrap/login.

### Carcasa global

- [ ] Cabecera con sección activa.
- [ ] Menú de usuario abierto/cerrado y usable por toque.
- [ ] Estados sano, caído y no configurado de cada integración.
- [ ] Acceso secundario a estados en móvil.
- [ ] Banda de suplantación y retorno al admin.
- [ ] Toast, error global y ruta no encontrada.

### Inicio y búsqueda

- [ ] Inicio con destacado, seguir viendo y rejilla mixta.
- [ ] Inicio sin progreso.
- [ ] Tarjeta disponible, parcial, no descargada y comprobando.
- [ ] Imagen/metadata ausente.
- [ ] Filtro Sólo descargado con y sin resultados.
- [ ] Búsqueda inicial, buscando, con resultados, sin resultados y error.
- [ ] Selector Películas/Series.
- [ ] Versiones escritorio y móvil.

### Biblioteca

- [ ] Lista mixta de películas y episodios.
- [ ] Item resuelto y accionable.
- [ ] Item sin match TMDB/no accionable.
- [ ] Biblioteca vacía, cargando y error.
- [ ] Path largo en móvil.

### Película y colección

- [ ] Película disponible con Reproducir/Reanudar.
- [ ] Película no disponible con torrent.
- [ ] Plarr degradado: sin Play/Download.
- [ ] Jackett/qBit degradados: sin Torrent.
- [ ] Estado visto/no visto y progreso.
- [ ] Dropdown de descarga con original y cinco renditions.
- [ ] Refresco de metadatos en curso/éxito/error.
- [ ] Colección y recomendaciones.
- [ ] Colección con partes y colección vacía.
- [ ] Hero sin artwork o sin sinopsis.
- [ ] Escritorio y móvil con todas las acciones alcanzables.

### Serie y episodios

- [ ] Serie sin disponibilidad, parcial y completa.
- [ ] Progreso visto separado de disponibilidad.
- [ ] Temporada cerrada/abierta, completa/parcial/ninguna.
- [ ] Episodio disponible, no disponible, visto y con progreso parcial.
- [ ] Acción Play/Reanudar y acción torrent por episodio.
- [ ] Fallo/reintento de temporada.
- [ ] Episodio objetivo enfocado tras navegación desde player.
- [ ] Lista de episodios adaptada a móvil.

### Reproducción

- [ ] Overlay cargando y reproduciendo.
- [ ] Buffering/conexión lenta y rendition muy baja.
- [ ] Calidad Auto y selección manual.
- [ ] Menús de audio y subtítulos.
- [ ] Reanudar desde posición guardada.
- [ ] Error “Playback unavailable” con salida/reintento.
- [ ] Botón cerrar y retorno al detalle.
- [ ] Anterior/siguiente.
- [ ] Cuenta atrás 5 s y Cancelar.
- [ ] Siguiente disponible, siguiente ausente y fin de serie.
- [ ] Fullscreen móvil vertical y horizontal.

### Torrent

- [ ] Modal inicial antes de buscar.
- [ ] Consulta editable y categoría prellenada/editable.
- [ ] Buscando, resultados y ningún resultado.
- [ ] Mínimo de seeds, trackers, seleccionar/limpiar, incluir y excluir.
- [ ] Todos filtrados como estado distinto.
- [ ] Fila lista, enviando, enviada y reintentable tras fallo.
- [ ] Confirmación persistente más toast.
- [ ] Error Jackett y error qBittorrent.
- [ ] Modal móvil con títulos largos y teclado abierto.

### Ajustes y administración

- [ ] Idioma en/es/pt/fr y aviso de fallback de UI.
- [ ] Guardando/guardado/error.
- [ ] Borrado de metadata con confirmación y éxito.
- [ ] Admin con formulario de alta y lista.
- [ ] Badge único de administrador.
- [ ] Diálogos renombrar, resetear contraseña y borrar.
- [ ] Validación, conflicto y error de cada acción.
- [ ] Suplantar y volver.
- [ ] Estado/configuración segura del servidor.
- [ ] Administración móvil.

### Calidad transversal

- [ ] Foco, teclado y cierre Escape en todos los overlays.
- [ ] Contraste sobre cada backdrop.
- [ ] Estados comprensibles sin depender del color.
- [ ] Textos largos y traducciones española/inglesa.
- [ ] 320 px, 390 px, tablet y escritorio ancho.
- [ ] Carga, vacío, error, éxito y degradación diferenciados.
- [ ] Ninguna captura promete features de la sección 14.

## 17. Criterio de aceptación del prototipo navegable

Un prototipo de capturas con zonas clicables conserva el producto sólo si permite recorrer sin callejones los siguientes caminos completos:

1. bootstrap/login → Inicio → búsqueda → detalle;
2. película no descargada → torrent → enviado → todavía pendiente → biblioteca disponible → play;
3. serie parcial → temporada → episodio disponible → player → siguiente no disponible → torrent;
4. película con progreso → reanudar → cambiar pista/calidad → cerrar → progreso conservado;
5. miembro → idioma → UI/metadatos localizados sin alterar nombres físicos;
6. admin → crear/editar/resetear/suplantar miembro → volver al admin;
7. cada combinación degradada de Plarr y Jackett/qBittorrent;
8. los mismos flujos críticos en móvil.

Cada área clicable debe tener un destino o cambio de estado explícito; si una acción no está implementada en el producto inicial, debe omitirse en vez de convertirse en un hotspot sin resultado.

## 18. Contrato de fixtures para prototipos y pruebas visuales

Los datos ficticios usados para validar la interfaz deben formar una única historia coherente entre capturas, estados y tamaños de viewport. No se deben mezclar especímenes incompatibles dentro del estado activo; carga, vacío, éxito y error pueden compartir una lámina de referencia, pero el prototipo debe tratarlos como variantes mutuamente excluyentes.

Fixture canónico de esta propuesta:

- `Órbita Ciega` recorre cuatro estados sucesivos: remota sin Play/Download; torrent aceptado pero todavía pendiente; archivo organizado y visible en Descargado; disponible con progreso personal conservado y acceso a Plarr.
- `Puerto Umbral` permanece parcial con `5/8` episodios en disco y `3/8` vistos. `S01E01` y `S01E02` están disponibles; `S01E03` está ausente y ofrece torrent. La rama de autoplay usa `S01E05 → S01E06`, ambos disponibles, para no contradecir la rama del episodio ausente.
- La biblioteca puede mostrar títulos localizados, pero todas las rutas de ejemplo usan nombres físicos canónicos ingleses, como `Movies/Blind Orbit (2026)/Blind Orbit (2026).mkv` y `Series/Threshold Harbor/Season 01/Threshold Harbor - S01E02.mkv`.
- Una sesión de player iniciada desde película conserva el contexto de película; una iniciada desde episodio conserva nombre de serie, temporada, episodio, navegación anterior/siguiente y retorno a la serie.
- La suplantación es un estado global del prototipo: su banda permanece sobre todas las pantallas y overlays hasta ejecutar “Volver al administrador”.
- En móvil, la navegación principal es siempre Inicio, Buscar y Descargado. La salud de servicios vive en un panel secundario uniforme; Plarr, Jackett y qBittorrent nunca se convierten en destinos principales.
- Las variantes degradadas deben obedecer la matriz de la sección 9: Plarr caído retira Play/Download; Jackett o qBittorrent caídos retiran Torrent; el resto de las acciones sanas permanece.

La validación del prototipo debe comprobar estas relaciones de forma cruzada, no sólo que cada imagen exista o que cada hotspot tenga un destino válido.
