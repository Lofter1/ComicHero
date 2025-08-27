/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_440763926")

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "date35049182",
    "max": "",
    "min": "",
    "name": "coverDate",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_440763926")

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "date35049182",
    "max": "",
    "min": "",
    "name": "releaseDate",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
})
