# ClassGame Web

Next.js teacher dashboard and Three.js student classroom.

- [Local setup](../docs/development/local-setup.md)
- [Configuration](../docs/reference/configuration.md)
- [Frontend module anatomy](../docs/architecture/modules.md#frontend)
- [Validation](../docs/development/validation.md)
- [Recover unsent student work](../docs/development/OFFLINE-EVIDENCE.md)

## Development

After configuring `.env.local` through the setup guide:

```bash
npm ci
npm run dev
```

Open [student game](http://localhost:3000/home) or [teacher login](http://localhost:3000/login). Installation synchronizes the Draco decoder. New routes stay in `src/app`; business behavior belongs to modules.

## Controls

| Action | Phone | Desktop |
| --- | --- | --- |
| Walk | Lower-screen joystick | WASD; Shift to run |
| Look | Swipe | Drag |
| Reach a desk | Tap its ring | Click its ring |
| Sit or stand | HUD buttons | HUD buttons |

## Reference

<a id="firebase"></a>
<a id="uploadcare"></a>

Authentication and optional uploads use the [configuration guide](../docs/reference/configuration.md). Keep private credentials server-side.

<a id="current-scope"></a>
<a id="structure"></a>
<a id="next-see-the-plan"></a>

See [system architecture](../docs/architecture/system.md) and the [implementation plan](../docs/plans/refactor-plan.md).

<a id="deploy-to-google-cloud-run"></a>
<a id="github-deploy-trigger"></a>

See [operations](../docs/operations/cloud-run.md). The former hosted deployment is offline.

## Assets

Runtime URLs under `public/` remain stable. Source [avatar credits/build notes](../assets/3d/avatars/chibi-band/README.md) and [worksheet notes](../assets/grade/README.md) remain beside their files.

## Credits

The [root credits](../README.md#credits) link to asset provenance and licenses.
